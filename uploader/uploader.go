package uploader

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	localLog "github.com/daqnext/LocalLog/log"
	"github.com/daqnext/go-smart-routine/sr"
	"github.com/olivere/elastic/v7"
)

const INDEX_TAGLOG = "infolog"
const INDEX_SQLDBLOG = "sqldblog"
const INDEX_GOLANG_PANIC = "errorlog"
const INDEX_JOBCURRENT = "jobcurrent"
const INDEX_UPLOAD_STATICS = "upload_statics"

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func (upl *Uploader) GenRandIdStr() string {
	x := randStr(42)
	return x
}

//if Id string field is ok ,return the trimed version of it
func checkStringIdField(Iface interface{}) (string, error) {
	ValueIface := reflect.ValueOf(Iface)

	// Check if the passed interface is a pointer ,must be a pointer ,otherwise error
	if ValueIface.Type().Kind() != reflect.Ptr {
		return "", errors.New("checkStringIdField not pointer error")
	}

	// 'dereference' with Elem() and get the field by name
	Field := ValueIface.Elem().FieldByName("Id")
	if !Field.IsValid() {
		return "", errors.New("error:Interface does not have the string Id field")
	}

	var str_value string
	typestr := Field.Type().String()
	switch {
	case typestr == "string":
		str_value = Field.String()
	case typestr == "int", typestr == "int64", typestr == "int8", typestr == "int16", typestr == "int32":
		str_value = strconv.FormatInt(Field.Int(), 10)
	default:
		return "", errors.New("err:Id type error only support int/int64/int16/int8")
	}

	trimvalue := strings.TrimSpace(str_value)
	if trimvalue == "" {
		return "", errors.New("error:Id string filed is vacant")
	}
	return trimvalue, nil
}

//batch upload every 50
const UPLOAD_DEFAULT_SIZE = 100

type InfoLog struct {
	Id       string
	App      string
	Tag      string
	Content  string
	Ip       string
	Instance string
	Time     int64
}

type SqldbLog struct {
	Id       string
	App      string
	Tag      string
	Content  string
	Ip       string
	Instance string
	Time     int64
}

type Errorlog struct {
	Id       string
	App      string
	Tag      string
	Content  string
	Ip       string
	Instance string
	Time     int64
}

type JobCurrent struct {
	Id       string
	App      string
	Name     string
	Tag      string
	Content  string
	Ip       string
	Instance string
	Time     int64
	Duration int64
}

type UploadStatics struct {
	Id       string
	Ip       string
	Instance string
	Time     int64
	Index    string
	Count    int
}

type Uploader struct {
	Client         *elastic.Client
	AnyLogs        map[string]map[string]interface{}
	AnyLogs_lock   sync.Mutex
	AnyLogsStarted sync.Map
	Ip             string
	InstanceIdStr  string
	llog           *localLog.LocalLog
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
			upl.AddErrorLog_Async(
				"",
				"uploader",
				"LogOverFlow:"+indexName,
				"!!error critical!! UserDefinedMapLogs:"+indexName+": length too big > 1000000",
				time.Now().Unix())
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
			return []string{}, iderr
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
func (upl *Uploader) AddInfoLog_Async(Instance string, App string, Tag string, Content string, Time int64) {
	upl.AddAnyLog_Async(INDEX_TAGLOG, &InfoLog{upl.GenRandIdStr(), App, Tag, Content, upl.GetPublicIP(), Instance, Time})
}

func (upl *Uploader) AddSqldbLog_Async(Instance string, App string, Tag string, Content string, Time int64) {
	upl.AddAnyLog_Async(INDEX_SQLDBLOG, &SqldbLog{upl.GenRandIdStr(), App, Tag, Content, upl.GetPublicIP(), Instance, Time})
}

func (upl *Uploader) AddErrorLog_Async(Instance string, App string, Tag string, Content string, Time int64) {
	data := []byte(App + Tag + Content)
	hashkey := fmt.Sprintf("%x", md5.Sum(data))
	upl.AddAnyLog_Async(INDEX_GOLANG_PANIC, &Errorlog{hashkey, App, Tag, Content, upl.GetPublicIP(), Instance, Time})
}

func (upl *Uploader) AddJobCurrent_Async(Instance string, App string, Name string, Tag string, Content string, Time int64, Duration int64) {
	data := []byte(App + ":" + Tag + ":" + Name + ":" + upl.GetPublicIP())
	hashkey := fmt.Sprintf("%x", md5.Sum(data))
	upl.AddAnyLog_Async(INDEX_JOBCURRENT, &JobCurrent{hashkey, App, Name, Tag, Content, upl.GetPublicIP(), Instance, Time, Duration})
}

func (upl *Uploader) GetPublicIP() string {
	return upl.Ip
}

func (upl *Uploader) GetInstanceId() string {
	return upl.InstanceIdStr
}

func New(endpoint string, username string, password string, llog_ *localLog.LocalLog) (*Uploader, error) {

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

	ipstr, errIp := GetPubIp()
	if errIp != nil {
		return nil, errIp
	}

	instanceIdstr := GetInstanceId()

	upl := &Uploader{
		Client:        client,
		Ip:            ipstr,
		InstanceIdStr: instanceIdstr,
		AnyLogs:       make(map[string]map[string]interface{}),
		llog:          llog_,
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
				}, upl.llog).Start()
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
					Id:       lmkindex + ":" + upl.GetPublicIP(),
					Instance: upl.GetInstanceId(),
					Ip:       upl.GetPublicIP(),
					Time:     time.Now().Unix(),
					Index:    lmkindex,
					Count:    len(upl.AnyLogs[lmkindex]),
				})
			}
			time.Sleep(300 * time.Second)
		}
	}, upl.llog).Start()
}
