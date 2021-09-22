# logservice
upload logs to remote elastic search server


## install 
```go
go get github.com/daqnext/logservice
```

## example
```go

package main

import (
	"time"

	"github.com/daqnext/logservice/uploader"
)

func main() {

	endpoint := "xxx"
	username := "yyy"
	password := "zzz"

	logUploader, err := uploader.New(endpoint, username, password)
	if err != nil {
		panic(err.Error())
	}

	///add job current
	logUploader.AddJobCurrent("TestApp", "job1", "jobcontent1 xxx", time.Now().Unix(), 123)
	logUploader.AddJobCurrent("TestApp", "job2", "jobcontent2 xxx", time.Now().Unix(), 12)
	logUploader.AddJobCurrent("TestApp", "job3", "jobcontent3 xxx", time.Now().Unix(), 0)
	logUploader.AddJobCurrent("TestApp", "job4", "jobcontent4 xxx", time.Now().Unix(), 1230)
	logUploader.AddJobCurrent("TestApp", "job1", "jobcontent1 yyy", time.Now().Unix(), 123)
	logUploader.AddJobCurrent("TestApp", "job2", "jobcontent2 yyy", time.Now().Unix(), 12)
	logUploader.AddJobCurrent("TestApp", "job3", "jobcontent3 yyy", time.Now().Unix(), 0)
	logUploader.AddJobCurrent("TestApp", "job4", "jobcontent4 yyy", time.Now().Unix(), 1230)
	logUploader.AddJobCurrent("TestApp", "job1", "jobcontent1 zzz", time.Now().Unix(), 123)
	logUploader.AddJobCurrent("TestApp", "job2", "jobcontent2 zzz", time.Now().Unix(), 12)
	logUploader.AddJobCurrent("TestApp", "job3", "jobcontent3 zzz", time.Now().Unix(), 0)
	logUploader.AddJobCurrent("TestApp", "job4", "jobcontent4 zzz", time.Now().Unix(), 1230)

	//add panic
	logUploader.AddGolangPanic("TestApp1", "panic test1", time.Now().Unix())
	logUploader.AddGolangPanic("TestApp1", "panic test2", time.Now().Unix())
	logUploader.AddGolangPanic("TestApp1", "panic test3", time.Now().Unix())
	logUploader.AddGolangPanic("TestApp2", "panic test1", time.Now().Unix())
	logUploader.AddGolangPanic("TestApp2", "panic test2", time.Now().Unix())
	logUploader.AddGolangPanic("TestApp1", "panic test3", time.Now().Unix())

	//add sql
	logUploader.AddSqldbLog("TestApp1", "sql insert xxxx 1", time.Now().Unix())
	logUploader.AddSqldbLog("TestApp1", "sql insert xxxx 2", time.Now().Unix())
	logUploader.AddSqldbLog("TestApp2", "sql insert xxxx 3", time.Now().Unix())
	logUploader.AddSqldbLog("TestApp3", "sql insert xxxx 4", time.Now().Unix())
	logUploader.AddSqldbLog("TestApp100", "sql insert xxxx 5", time.Now().Unix())

	//add customlog
	logUploader.AddCustomLog("tag1", "this is custom message 1", time.Now().Unix())
	logUploader.AddCustomLog("tag2", "this is custom message 2", time.Now().Unix())
	logUploader.AddCustomLog("tag1", "this is custom message 1", time.Now().Unix())

	time.Sleep(time.Second * 600)

}

```