package uploader

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"sync"
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
	CurrentJobs               sync.Map
	GolangPanics              map[string]*GolangPanic
	GolangPanics_lock         sync.Mutex
	SqldbLogs                 []*SqldbLog
	SqldbLogs_lock            sync.Mutex
	TagLogs                   []*TagLog
	TagLogs_lock              sync.Mutex
	UserDefinedLogs           map[string][]interface{}
	UserDefinedLogs_lock      sync.Mutex
	UserDefinedLogsStarted    sync.Map
	UserDefinedMapLogs        map[string]map[string]interface{}
	UserDefinedMapLogs_lock   sync.Mutex
	UserDefinedMapLogsStarted sync.Map
	Ip                        string
}

////////UserDefinedLogs///////////

func (upl *Uploader) AddUserDefinedLog(indexName string, log interface{}) {
	upl.UserDefinedLogs_lock.Lock()

	_, ok := upl.UserDefinedLogs[indexName]
	if !ok {
		upl.UserDefinedLogs[indexName] = []interface{}{}
	}
	upl.UserDefinedLogs[indexName] = append(upl.UserDefinedLogs[indexName], log)

	upl.UserDefinedLogs_lock.Unlock()
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
				upl.UserDefinedLogs_lock.Lock()
				upl.UserDefinedLogs[indexName] = upl.UserDefinedLogs[indexName][toUploadSize:]
				upl.UserDefinedLogs_lock.Unlock()
			}
		}

		//delete some record as too much records
		if len(upl.UserDefinedLogs[indexName]) > 1000000 {
			upl.AddGolangPanic("uploader", "UserDefinedLogs:"+indexName+": length too big > 1000000", time.Now().Unix())
			upl.UserDefinedLogs_lock.Lock()
			upl.UserDefinedLogs[indexName] = upl.UserDefinedLogs[indexName][900000:]
			upl.UserDefinedLogs_lock.Unlock()
		}

		//wait and add
		if len(upl.UserDefinedLogs[indexName]) < 10 {
			time.Sleep(120 * time.Second)
		}
	}

}

func (upl *Uploader) AddUserDefinedMapLog(indexName string, mapkey string, log interface{}) {

	upl.UserDefinedMapLogs_lock.Lock()

	_, ok := upl.UserDefinedMapLogs[indexName]
	if !ok {
		upl.UserDefinedMapLogs[indexName] = make(map[string]interface{})
	}
	upl.UserDefinedMapLogs[indexName][mapkey] = log

	upl.UserDefinedMapLogs_lock.Unlock()
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
				upl.UserDefinedMapLogs_lock.Lock()
				for i := 0; i < len(keysToUpload); i++ {
					delete(upl.UserDefinedMapLogs[indexName], keysToUpload[i])
				}
				upl.UserDefinedMapLogs_lock.Unlock()
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
	upl.TagLogs_lock.Lock()
	upl.TagLogs = append(upl.TagLogs, &TagLog{App, Tag, Content, upl.Ip, Time})
	upl.TagLogs_lock.Unlock()
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
				upl.TagLogs_lock.Lock()
				upl.TagLogs = upl.TagLogs[toUploadSize:]
				upl.TagLogs_lock.Unlock()
			}
		}

		//delete some record as too much records
		if len(upl.TagLogs) > 1000000 {
			upl.AddGolangPanic("uploader", "TagLogs length too big > 1000000", time.Now().Unix())
			upl.TagLogs_lock.Lock()
			upl.TagLogs = upl.TagLogs[900000:]
			upl.TagLogs_lock.Unlock()
		}

		//wait and add
		if len(upl.TagLogs) < 10 {
			time.Sleep(120 * time.Second)
		}
	}

}

func (upl *Uploader) AddSqldbLog(App string, Content string, Time int64) {
	upl.SqldbLogs_lock.Lock()
	upl.SqldbLogs = append(upl.SqldbLogs, &SqldbLog{App, Content, upl.Ip, Time})
	upl.SqldbLogs_lock.Unlock()
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
				upl.SqldbLogs_lock.Lock()
				upl.SqldbLogs = upl.SqldbLogs[toUploadSize:]
				upl.SqldbLogs_lock.Unlock()
			}
		}

		//delete some record as too much records
		if len(upl.SqldbLogs) > 1000000 {
			upl.AddGolangPanic("uploader", "SqldbLog length too big > 1000000", time.Now().Unix())
			upl.SqldbLogs_lock.Lock()
			upl.SqldbLogs = upl.SqldbLogs[900000:]
			upl.SqldbLogs_lock.Unlock()
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
	upl.GolangPanics_lock.Lock()
	upl.GolangPanics[hashkey] = &GolangPanic{App, Content, upl.Ip, Time}
	upl.GolangPanics_lock.Unlock()
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
				upl.GolangPanics_lock.Lock()
				for i := 0; i < len(keysToUpload); i++ {
					delete(upl.GolangPanics, keysToUpload[i])
				}
				upl.GolangPanics_lock.Unlock()
			}
		}

		//delete some record as too much records
		if len(upl.GolangPanics) > 1000000 {
			//not going to happen actually
			upl.GolangPanics_lock.Lock()
			upl.GolangPanics = make(map[string]*GolangPanic)
			upl.GolangPanics_lock.Unlock()
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
	upl.CurrentJobs.Store(hashkey, &JobCurrent{App, Name, Content, upl.Ip, Time, Duration})
}

func (upl *Uploader) uploadJobCurrent() {

	bulkRequest := upl.Client.Bulk()
	for {

		jobsCounter := 0
		upl.CurrentJobs.Range(func(_, v interface{}) bool {
			reqi := elastic.NewBulkIndexRequest().Index("jobcurrent").Doc(v).Id(v.(*JobCurrent).App + v.(*JobCurrent).Name)
			bulkRequest.Add(reqi)
			jobsCounter++
			return true
		})

		if jobsCounter > 0 {
			_, err := bulkRequest.Do(context.Background())
			if err != nil {
				upl.AddGolangPanic("uploader", "uploadJobCurrent:"+err.Error(), time.Now().Unix())
				time.Sleep(120 * time.Second)
				continue
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

	upl := &Uploader{
		Client:             client,
		GolangPanics:       make(map[string]*GolangPanic),
		SqldbLogs:          []*SqldbLog{},
		TagLogs:            []*TagLog{},
		UserDefinedLogs:    make(map[string][]interface{}),
		UserDefinedMapLogs: make(map[string]map[string]interface{}),
		Ip:                 GetPubIp(),
	}

	upl.start()
	return upl, nil
}

func (upl *Uploader) StartUserDefinedLogsUpload() {

	for {
		for lkindex, _ := range upl.UserDefinedLogs {
			_, ok := upl.UserDefinedLogsStarted.Load(lkindex)
			if !ok {
				upl.UserDefinedLogsStarted.Store(lkindex, true)
				sr.New_Panic_RedoWithContext(lkindex, func(indexstr interface{}) {
					upl.uploadUserDefinedLog(indexstr.(string))
				}).Start()
			}
		}

		for lmkindex, _ := range upl.UserDefinedMapLogs {
			_, ok := upl.UserDefinedMapLogsStarted.Load(lmkindex)
			if !ok {
				upl.UserDefinedMapLogsStarted.Store(lmkindex, true)
				sr.New_Panic_RedoWithContext(lmkindex, func(indexstr interface{}) {
					upl.uploadUserDefinedMapLog(indexstr.(string))
				}).Start()
			}
		}
		time.Sleep(5 * time.Second)
	}

}

func (upl *Uploader) start() {
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
