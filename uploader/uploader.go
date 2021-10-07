package uploader

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/daqnext/go-smart-routine/sr"
	"github.com/olivere/elastic/v7"
)

const INDEX_TAGLOG = "taglog"
const INDEX_SQLDBLOG = "sqldblog"
const INDEX_GOLANG_PANIC = "golang_panic"
const INDEX_JOBCURRENT = "jobcurrent"
const INDEX_UPLOAD_STATICS = "upload_statics"

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func randStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func (upl *Uploader) GenRandIdStr() string {
	return randStr(42)
}

//if Id string field is ok ,return the trimed version of it
func checkStringIdField(Iface interface{}) (string, error) {
	ValueIface := reflect.ValueOf(Iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface's Type, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(Iface))
	}

	// 'dereference' with Elem() and get the field by name
	Field := ValueIface.Elem().FieldByName("Id")
	if !Field.IsValid() {
		return "", errors.New("error:Interface does not have the string Id field")
	}
	value := Field.String()
	trimvalue := strings.TrimSpace(value)
	if trimvalue == "" {
		return "", errors.New("error:Id string filed is vacant")
	}
	return trimvalue, nil
}

//batch upload every 50
const UPLOAD_DEFAULT_SIZE = 100

type TagLog struct {
	Id      string
	App     string
	Tag     string
	Content string
	Ip      string
	Time    int64
}

type SqldbLog struct {
	Id      string
	App     string
	Content string
	Ip      string
	Time    int64
}

type GolangPanic struct {
	Id      string
	App     string
	Content string
	Ip      string
	Time    int64
}

type JobCurrent struct {
	Id       string
	App      string
	Name     string
	Content  string
	Ip       string
	Time     int64
	Duration int64
}

type UploadStatics struct {
	Id    string
	Ip    string
	Time  int64
	Index string
	Count int
}

type Uploader struct {
	Client         *elastic.Client
	AnyLogs        map[string]map[string]interface{}
	AnyLogs_lock   sync.Mutex
	AnyLogsStarted sync.Map
	Ip             string
}

func (upl *Uploader) AddAnyLog_Async(indexName string, log interface{}) error {

	idstr, iderr := checkStringIdField(log)
	if iderr != nil {
		return iderr
	}

	upl.AnyLogs_lock.Lock()
	_, ok := upl.AnyLogs[indexName]
	if !ok {
		upl.AnyLogs[indexName] = make(map[string]interface{})
	}
	upl.AnyLogs[indexName][idstr] = log
	upl.AnyLogs_lock.Unlock()
	return nil
}

func (upl *Uploader) uploadAnyLog_Async(indexName string) {

	bulkRequest := upl.Client.Bulk()

	for {

		toUploadSize := 0
		for k, v := range upl.AnyLogs[indexName] {
			if toUploadSize >= UPLOAD_DEFAULT_SIZE {
				break
			}
			toUploadSize++
			reqi := elastic.NewBulkIndexRequest().Index(indexName).Doc(v).Id(k)
			bulkRequest.Add(reqi)
		}

		if toUploadSize > 0 {
			bulkresponse, _ := bulkRequest.Do(context.Background())
			if len(bulkresponse.Failed()) > 0 {
				time.Sleep(30 * time.Second)
			}
			successList := bulkresponse.Succeeded()
			upl.AnyLogs_lock.Lock()
			for i := 0; i < len(successList); i++ {
				delete(upl.AnyLogs[indexName], successList[i].Id)
			}
			upl.AnyLogs_lock.Unlock()
		}

		//give warnings to system
		if len(upl.AnyLogs[indexName]) > 1000000 {
			upl.AddGolangPanic_Async("uploader", "!!error critical!! UserDefinedMapLogs:"+indexName+": length too big > 1000000", time.Now().Unix())
		}

		//wait and add
		if len(upl.AnyLogs[indexName]) < 10 {
			time.Sleep(120 * time.Second)
		}
	}

}

func (upl *Uploader) UploadAnyLogs_Sync(indexName string, logs []interface{}) (succeededIds []string, idInputError error) {
	if len(logs) == 0 {
		return []string{}, nil
	}
	bulkRequest := upl.Client.Bulk()
	for i := 0; i < len(logs); i++ {
		idstr, iderr := checkStringIdField(logs[i])
		if iderr != nil {
			return []string{}, errors.New("no Id field")
		}
		reqi := elastic.NewBulkIndexRequest().Index(indexName).Doc(logs[i]).Id(idstr)
		bulkRequest.Add(reqi)
	}
	resp, _ := bulkRequest.Do(context.Background())
	succeeded := resp.Succeeded()
	for i := 0; i < len(succeeded); i++ {
		succeededIds = append(succeededIds, succeeded[i].Id)
	}
	return succeededIds, nil
}

//////end of UserDefinedLogs/////////////
func (upl *Uploader) AddTagLog_Async(App string, Tag string, Content string, Time int64) {
	upl.AddAnyLog_Async(INDEX_TAGLOG, &TagLog{upl.GenRandIdStr(), App, Tag, Content, upl.GetPublicIP(), Time})
}

func (upl *Uploader) AddSqldbLog_Async(App string, Content string, Time int64) {
	upl.AddAnyLog_Async(INDEX_SQLDBLOG, &SqldbLog{upl.GenRandIdStr(), App, Content, upl.GetPublicIP(), Time})
}

func (upl *Uploader) AddGolangPanic_Async(App string, Content string, Time int64) {
	data := []byte(App + Content)
	hashkey := fmt.Sprintf("%x", md5.Sum(data))
	upl.AddAnyLog_Async(INDEX_GOLANG_PANIC, &GolangPanic{hashkey, App, Content, upl.GetPublicIP(), Time})
}

func (upl *Uploader) AddJobCurrent_Async(App string, Name string, Content string, Time int64, Duration int64) {
	hashkey := App + ":" + Name + ":" + upl.GetPublicIP()
	upl.AddAnyLog_Async(INDEX_JOBCURRENT, &JobCurrent{hashkey, App, Name, Content, upl.GetPublicIP(), Time, Duration})
}

func (upl *Uploader) GetPublicIP() string {
	return upl.Ip
}

func New(endpoint string, username string, password string) (*Uploader, error) {

	client, err := elastic.NewClient(
		elastic.SetURL(endpoint),
		elastic.SetBasicAuth(username, password),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(30*time.Second),
		//elastic.SetRetrier(&myRetrier{}), //"never use retrier as bad request should be handled by client" by leo
		elastic.SetGzip(true),
	)
	if err != nil {
		return nil, err
	}

	upl := &Uploader{
		Client:  client,
		Ip:      GetPubIp(),
		AnyLogs: make(map[string]map[string]interface{}),
	}

	upl.start()
	return upl, nil
}

func (upl *Uploader) startAnyLogsUpload() {
	for {
		for lmkindex, _ := range upl.AnyLogs {
			_, ok := upl.AnyLogsStarted.Load(lmkindex)
			if !ok {
				upl.AnyLogsStarted.Store(lmkindex, true)
				sr.New_Panic_RedoWithContext(lmkindex, func(indexstr interface{}) {
					upl.uploadAnyLog_Async(indexstr.(string))
				}).Start()
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func (upl *Uploader) start() {
	//start the userdefined
	go upl.startAnyLogsUpload()
	//upload statics monitor
	sr.New_Panic_Redo(func() {
		for {
			for lmkindex, _ := range upl.AnyLogs {
				upl.AddAnyLog_Async(INDEX_UPLOAD_STATICS, &UploadStatics{
					Id:    lmkindex + ":" + upl.GetPublicIP(),
					Ip:    upl.GetPublicIP(),
					Time:  time.Now().Unix(),
					Index: lmkindex,
					Count: len(upl.AnyLogs[lmkindex]),
				})
			}
			time.Sleep(300 * time.Second)
		}
	}).Start()
}
