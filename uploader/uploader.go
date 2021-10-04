package uploader

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"github.com/daqnext/go-smart-routine/sr"
	"github.com/olivere/elastic/v7"
)

//batch upload every 50
const UPLOAD_DEFAULT_SIZE = 100

type TagLog struct {
	App     string `json:"app"`
	Tag     string `json:"tag"`
	Content string `json:"content"`
	Ip      string `json:"ip"`
	Time    int64  `json:"time"`
}

type SqldbLog struct {
	App     string `json:"app"`
	Content string `json:"content"`
	Ip      string `json:"ip"`
	Time    int64  `json:"time"`
}

type GolangPanic struct {
	App     string `json:"app"`
	Content string `json:"content"`
	Ip      string `json:"ip"`
	Time    int64  `json:"time"`
}

type JobCurrent struct {
	App      string `json:"app"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	Ip       string `json:"ip"`
	Time     int64  `json:"time"`
	Duration int64  `json:"duration"`
}

type myRetrier struct {
}

func (r *myRetrier) Retry(ctx context.Context, retry int, req *http.Request, resp *http.Response, err error) (time.Duration, bool, error) {
	return 300 * time.Second, true, nil //retry after 5mins
}

type Uploader struct {
	Client                    *elastic.Client
	CurrentJobs               map[string]*JobCurrent
	GolangPanics              map[string]*GolangPanic
	SqldbLogs                 []*SqldbLog
	TagLogs                   []*TagLog
	UserDefinedLogs           map[string][]interface{}
	UserDefinedLogsStarted    map[string]bool
	UserDefinedMapLogs        map[string]map[string]interface{}
	UserDefinedMapLogsStarted map[string]bool
	Ip                        string
}

////////UserDefinedLogs///////////

func (upl *Uploader) AddUserDefinedLog(indexName string, log interface{}) {
	upl.UserDefinedLogs[indexName] = append(upl.UserDefinedLogs[indexName], log)
}

func (upl *Uploader) uploadUserDefinedLog(indexName string) {
	bulkRequest := upl.Client.Bulk()

	for {

		toUploadSize := UPLOAD_DEFAULT_SIZE
		if toUploadSize > len(upl.UserDefinedLogs[indexName]) {
			toUploadSize = len(upl.UserDefinedLogs[indexName])
		}

		if toUploadSize > 0 {
			for i := 0; i < toUploadSize; i++ {
				reqi := elastic.NewBulkIndexRequest().Index(indexName).Doc(upl.UserDefinedLogs[indexName][i])
				bulkRequest.Add(reqi)
			}

			_, err := bulkRequest.Do(context.Background())
			if err != nil {
				time.Sleep(120 * time.Second)
			} else {
				upl.UserDefinedLogs[indexName] = upl.UserDefinedLogs[indexName][toUploadSize:]
			}
		}

		//delete some record as too much records
		if len(upl.UserDefinedLogs[indexName]) > 1000000 {
			upl.AddGolangPanic("uploader", "UserDefinedLogs:"+indexName+": length too big > 1000000", time.Now().Unix())
			upl.UserDefinedLogs[indexName] = upl.UserDefinedLogs[indexName][900000:]
		}

		//wait and add
		if len(upl.UserDefinedLogs[indexName]) < 10 {
			time.Sleep(120 * time.Second)
		}
	}

}

func (upl *Uploader) AddUserDefinedMapLog(indexName string, mapkey string, log interface{}) {
	if upl.UserDefinedMapLogs[indexName] == nil {
		upl.UserDefinedMapLogs[indexName] = make(map[string]interface{})
	}

	upl.UserDefinedMapLogs[indexName][mapkey] = log
}

func (upl *Uploader) uploadUserDefinedMapLog(indexName string) {

	bulkRequest := upl.Client.Bulk()

	for {

		toUploadSize := 0
		keysToUpload := []string{}

		for k, v := range upl.UserDefinedMapLogs[indexName] {
			if toUploadSize >= UPLOAD_DEFAULT_SIZE {
				break
			}
			toUploadSize++
			keysToUpload = append(keysToUpload, k)
			reqi := elastic.NewBulkIndexRequest().Index(indexName).Doc(v).Id(k)
			bulkRequest.Add(reqi)
		}

		if toUploadSize > 0 {
			_, err := bulkRequest.Do(context.Background())
			if err != nil {
				time.Sleep(120 * time.Second)
			} else {
				for i := 0; i < len(keysToUpload); i++ {
					delete(upl.UserDefinedMapLogs[indexName], keysToUpload[i])
				}
			}
		}

		//give warnings to system
		if len(upl.UserDefinedMapLogs[indexName]) > 1000000 {
			upl.AddGolangPanic("uploader", "!!error critical!! UserDefinedMapLogs:"+indexName+": length too big > 1000000", time.Now().Unix())
		}

		//wait and add
		if len(upl.UserDefinedMapLogs[indexName]) < 10 {
			time.Sleep(120 * time.Second)
		}
	}

}

//////end of UserDefinedLogs/////////////

func (upl *Uploader) AddTagLog(App string, Tag string, Content string, Time int64) {
	upl.TagLogs = append(upl.TagLogs, &TagLog{App, Tag, Content, upl.Ip, Time})
}

func (upl *Uploader) uploadTagLog() {

	bulkRequest := upl.Client.Bulk()

	for {

		toUploadSize := UPLOAD_DEFAULT_SIZE
		if toUploadSize > len(upl.TagLogs) {
			toUploadSize = len(upl.TagLogs)
		}

		if toUploadSize > 0 {
			for i := 0; i < toUploadSize; i++ {
				reqi := elastic.NewBulkIndexRequest().Index("taglog").Doc(upl.TagLogs[i])
				bulkRequest.Add(reqi)
			}

			_, err := bulkRequest.Do(context.Background())
			if err != nil {
				time.Sleep(120 * time.Second)
			} else {
				upl.TagLogs = upl.TagLogs[toUploadSize:]
			}
		}

		//delete some record as too much records
		if len(upl.TagLogs) > 1000000 {
			upl.AddGolangPanic("uploader", "TagLogs length too big > 1000000", time.Now().Unix())
			upl.TagLogs = upl.TagLogs[900000:]
		}

		//wait and add
		if len(upl.TagLogs) < 10 {
			time.Sleep(120 * time.Second)
		}
	}

}

func (upl *Uploader) AddSqldbLog(App string, Content string, Time int64) {
	upl.SqldbLogs = append(upl.SqldbLogs, &SqldbLog{App, Content, upl.Ip, Time})
}

func (upl *Uploader) uploadSqldbLog() {

	bulkRequest := upl.Client.Bulk()

	for {

		toUploadSize := UPLOAD_DEFAULT_SIZE
		if toUploadSize > len(upl.SqldbLogs) {
			toUploadSize = len(upl.SqldbLogs)
		}

		if toUploadSize > 0 {
			for i := 0; i < toUploadSize; i++ {
				reqi := elastic.NewBulkIndexRequest().Index("sqldblog").Doc(upl.SqldbLogs[i])
				bulkRequest.Add(reqi)
			}

			_, err := bulkRequest.Do(context.Background())
			if err != nil {
				time.Sleep(120 * time.Second)
			} else {
				upl.SqldbLogs = upl.SqldbLogs[toUploadSize:]
			}
		}

		//delete some record as too much records
		if len(upl.SqldbLogs) > 1000000 {
			upl.AddGolangPanic("uploader", "SqldbLog length too big > 1000000", time.Now().Unix())
			upl.SqldbLogs = upl.SqldbLogs[900000:]
		}

		//wait and add
		if len(upl.SqldbLogs) < 10 {
			time.Sleep(120 * time.Second)
		}
	}

}

func (upl *Uploader) AddGolangPanic(App string, Content string, Time int64) {

	data := []byte(App + Content)
	hashkey := fmt.Sprintf("%x", md5.Sum(data))
	upl.GolangPanics[hashkey] = &GolangPanic{App, Content, upl.Ip, Time}
}

func (upl *Uploader) uploadGolangPanic() {

	bulkRequest := upl.Client.Bulk()

	for {

		toUploadSize := 0
		keysToUpload := []string{}

		for k, v := range upl.GolangPanics {
			if toUploadSize >= UPLOAD_DEFAULT_SIZE {
				break
			}
			toUploadSize++
			keysToUpload = append(keysToUpload, k)
			reqi := elastic.NewBulkIndexRequest().Index("golang_panic").Doc(v).Id(k)
			bulkRequest.Add(reqi)
		}

		if toUploadSize > 0 {
			_, err := bulkRequest.Do(context.Background())
			if err != nil {
				time.Sleep(120 * time.Second)
			} else {
				for i := 0; i < len(keysToUpload); i++ {
					delete(upl.GolangPanics, keysToUpload[i])
				}
			}
		}

		//delete some record as too much records
		if len(upl.GolangPanics) > 1000000 {
			//not going to happen actually
			upl.GolangPanics = make(map[string]*GolangPanic)
			upl.AddGolangPanic("uploader", "!!error critical!! GolangPanics length too big > 1000000", time.Now().Unix())
		}

		//wait and add
		if len(upl.GolangPanics) < 10 {
			time.Sleep(120 * time.Second)
		}
	}

}

func (upl *Uploader) AddJobCurrent(App string, Name string, Content string, Time int64, Duration int64) {
	data := []byte(App + Name)
	hashkey := fmt.Sprintf("%x", md5.Sum(data))
	upl.CurrentJobs[hashkey] = &JobCurrent{App, Name, Content, upl.Ip, Time, Duration}
}

func (upl *Uploader) uploadJobCurrent() {

	bulkRequest := upl.Client.Bulk()
	for {
		for _, v := range upl.CurrentJobs {
			reqi := elastic.NewBulkIndexRequest().Index("jobcurrent").Doc(v).Id(v.App + v.Name)
			bulkRequest.Add(reqi)
		}

		if len(upl.CurrentJobs) > 0 {
			_, err := bulkRequest.Do(context.Background())
			if err != nil {
				time.Sleep(120 * time.Second)
				continue
			} else {
				upl.CurrentJobs = make(map[string]*JobCurrent)
			}
		}
		//wait and add
		time.Sleep(120 * time.Second)
	}

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
		elastic.SetRetrier(&myRetrier{}),
		elastic.SetGzip(true),
	)
	if err != nil {
		return nil, err
	}

	upl := &Uploader{client, make(map[string]*JobCurrent), make(map[string]*GolangPanic), []*SqldbLog{}, []*TagLog{},
		make(map[string][]interface{}), make(map[string]bool), make(map[string]map[string]interface{}), make(map[string]bool), GetPubIp()}

	return upl, nil
}

func (upl *Uploader) StartUserDefinedLogsUpload() {
	for {

		for lkindex, _ := range upl.UserDefinedLogs {
			if !upl.UserDefinedLogsStarted[lkindex] {
				upl.UserDefinedLogsStarted[lkindex] = true
				sr.New_Panic_Redo(func() {
					upl.uploadUserDefinedLog(lkindex)
				}).Start()
			}
		}

		for lmkindex, _ := range upl.UserDefinedMapLogs {
			if !upl.UserDefinedMapLogsStarted[lmkindex] {
				upl.UserDefinedMapLogsStarted[lmkindex] = true
				sr.New_Panic_Redo(func() {
					upl.uploadUserDefinedMapLog(lmkindex)
				}).Start()
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func (upl *Uploader) Start() {
	//start the backgroud uploader
	sr.New_Panic_Redo(func() {
		upl.uploadSqldbLog()
	}).Start()

	sr.New_Panic_Redo(func() {
		upl.uploadGolangPanic()
	}).Start()

	sr.New_Panic_Redo(func() {
		upl.uploadJobCurrent()
	}).Start()

	sr.New_Panic_Redo(func() {
		upl.uploadTagLog()
	}).Start()

	//start the userdefined
	go upl.StartUserDefinedLogsUpload()

}
