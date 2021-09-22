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

type CustomLog struct {
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
	Client       *elastic.Client
	CurrentJobs  map[string]*JobCurrent
	GolangPanics map[string]*GolangPanic
	SqldbLog     []*SqldbLog
	CustomLogs   []*CustomLog
	Ip           string
}

func (upl *Uploader) AddCustomLog(App string, Tag string, Content string, Time int64) {
	upl.CustomLogs = append(upl.CustomLogs, &CustomLog{App, Tag, Content, upl.Ip, Time})
}

func (upl *Uploader) uploadCustomLog() {

	bulkRequest := upl.Client.Bulk()

	for {

		toUploadSize := UPLOAD_DEFAULT_SIZE
		if toUploadSize > len(upl.CustomLogs) {
			toUploadSize = len(upl.CustomLogs)
		}

		if toUploadSize > 0 {
			for i := 0; i < toUploadSize; i++ {
				reqi := elastic.NewBulkIndexRequest().Index("customlog").Doc(upl.CustomLogs[i])
				bulkRequest.Add(reqi)
			}

			_, err := bulkRequest.Do(context.Background())
			if err != nil {
				time.Sleep(120 * time.Second)
			} else {
				upl.CustomLogs = upl.CustomLogs[toUploadSize:]
			}
		}

		//delete some record as too much records
		if len(upl.CustomLogs) > 100000 {
			upl.AddGolangPanic("uploader", "CustomLogs length too big > 100000", time.Now().Unix())
			upl.CustomLogs = upl.CustomLogs[50000:]
		}

		//wait and add
		if len(upl.CustomLogs) < 10 {
			time.Sleep(300 * time.Second)
		}
	}

}

func (upl *Uploader) AddSqldbLog(App string, Content string, Time int64) {
	upl.SqldbLog = append(upl.SqldbLog, &SqldbLog{App, Content, upl.Ip, Time})
}

func (upl *Uploader) uploadSqldbLog() {

	bulkRequest := upl.Client.Bulk()

	for {

		toUploadSize := UPLOAD_DEFAULT_SIZE
		if toUploadSize > len(upl.SqldbLog) {
			toUploadSize = len(upl.SqldbLog)
		}

		if toUploadSize > 0 {
			for i := 0; i < toUploadSize; i++ {
				reqi := elastic.NewBulkIndexRequest().Index("sqldblog").Doc(upl.SqldbLog[i])
				bulkRequest.Add(reqi)
			}

			_, err := bulkRequest.Do(context.Background())
			if err != nil {
				time.Sleep(120 * time.Second)
			} else {
				upl.SqldbLog = upl.SqldbLog[toUploadSize:]
			}
		}

		//delete some record as too much records
		if len(upl.SqldbLog) > 100000 {
			upl.AddGolangPanic("uploader", "SqldbLog length too big > 100000", time.Now().Unix())
			upl.SqldbLog = upl.SqldbLog[50000:]
		}

		//wait and add
		if len(upl.SqldbLog) < 10 {
			time.Sleep(300 * time.Second)
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
			toUploadSize++
			keysToUpload = append(keysToUpload, k)
			reqi := elastic.NewBulkIndexRequest().Index("golang_panic").Doc(v)
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
		if len(upl.GolangPanics) > 100000 {
			//not going to happen actually
			upl.GolangPanics = make(map[string]*GolangPanic)
			upl.AddGolangPanic("uploader", "!!error critical!! GolangPanics length too big > 100000", time.Now().Unix())
		}

		//wait and add
		if len(upl.GolangPanics) < 10 {
			time.Sleep(300 * time.Second)
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
		time.Sleep(300 * time.Second)
	}

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

	upl := &Uploader{client, make(map[string]*JobCurrent), make(map[string]*GolangPanic), []*SqldbLog{}, []*CustomLog{}, GetPubIp()}

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
		upl.uploadCustomLog()
	}).Start()

	return upl, nil
}
