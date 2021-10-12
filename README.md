# loguploader
server side upload logs to remote elastic search server

## install 
```go
go get github.com/daqnext/ESUploader
```

## example
```go


package main

import (
	"fmt"
	"time"

	"github.com/daqnext/ESUploader/uploader"
)

type UserDefinedLog struct {
	Id     int
	Field1 string
	Field2 int
	Ip     string
}

func main() {

	endpoint := "xxxx"
	username := "yyyy"
	password := "zzzz"

	ESUploader, err := uploader.New(endpoint, username, password)
	if err != nil {
		panic(err.Error())
	}

	udl1 := UserDefinedLog{123123, "hello", 1, ESUploader.GetPublicIP()}
	udl2 := UserDefinedLog{451324134, "world", 2, ESUploader.GetPublicIP()}
	udl3 := UserDefinedLog{77777, "hello world", 3, ESUploader.GetPublicIP()}

	succeededIds, errors := ESUploader.UploadAnyLogs_Sync("userdefinedlog", []interface{}{&udl1, &udl2, &udl3})

	fmt.Println(errors)
	fmt.Println(succeededIds)

	//return

	///add job current
	ESUploader.AddJobCurrent_Async(ESUploader.GetInstanceId(), "TestApp", "job1", "jobcontent1 xxx", time.Now().Unix(), 123)
	ESUploader.AddJobCurrent_Async("", "TestApp", "job2", "jobcontent2 xxx", time.Now().Unix(), 12)
	ESUploader.AddJobCurrent_Async("", "TestApp", "job3", "jobcontent3 xxx", time.Now().Unix(), 0)
	ESUploader.AddJobCurrent_Async("", "TestApp", "job4", "jobcontent4 xxx", time.Now().Unix(), 1230)
	ESUploader.AddJobCurrent_Async(ESUploader.GetInstanceId(), "TestApp", "job1", "jobcontent1 yyy", time.Now().Unix(), 123)
	ESUploader.AddJobCurrent_Async(ESUploader.GetInstanceId(), "TestApp", "job2", "jobcontent2 yyy", time.Now().Unix(), 12)
	ESUploader.AddJobCurrent_Async(ESUploader.GetInstanceId(), "TestApp", "job3", "jobcontent3 yyy", time.Now().Unix(), 0)
	ESUploader.AddJobCurrent_Async("", "TestApp", "job4", "jobcontent4 yyy", time.Now().Unix(), 1230)

	//add panic
	ESUploader.AddErrorLog_Async("", "TestApp1", "panic test1", time.Now().Unix())
	ESUploader.AddErrorLog_Async("", "TestApp1", "panic test2", time.Now().Unix())
	ESUploader.AddErrorLog_Async("", "TestApp1", "panic test3", time.Now().Unix())
	ESUploader.AddErrorLog_Async(ESUploader.GetInstanceId(), "TestApp2", "panic test1", time.Now().Unix())
	ESUploader.AddErrorLog_Async("", "TestApp2", "panic test2", time.Now().Unix())
	ESUploader.AddErrorLog_Async("", "TestApp1", "panic test3", time.Now().Unix())

	//add sql
	ESUploader.AddSqldbLog_Async("", "TestApp1", "sql insert xxxx 1", time.Now().Unix())
	ESUploader.AddSqldbLog_Async("", "TestApp1", "sql insert xxxx 2", time.Now().Unix())
	ESUploader.AddSqldbLog_Async(ESUploader.GetInstanceId(), "TestApp2", "sql insert xxxx 3", time.Now().Unix())
	ESUploader.AddSqldbLog_Async("", "TestApp3", "sql insert xxxx 4", time.Now().Unix())
	ESUploader.AddSqldbLog_Async("", "TestApp9", "sql insert xxxx 5", time.Now().Unix())

	//add taglog
	ESUploader.AddInfoLog_Async("", "App1", "tag1", "this is custom message 1", time.Now().Unix())
	ESUploader.AddInfoLog_Async("", "App2", "tag2", "this is custom message 2", time.Now().Unix())
	ESUploader.AddInfoLog_Async("", "App3", "tag1", "this is custom message 1", time.Now().Unix())

	//userdefined log

	udl1add := UserDefinedLog{333, "this is filed1", 2, ESUploader.GetPublicIP()}
	udl2add := UserDefinedLog{444, "this is filed2", 2, ESUploader.GetPublicIP()}
	udl3add := UserDefinedLog{555, "this is filed1", 2, ESUploader.GetPublicIP()}

	ESUploader.AddAnyLog_Async("userdefinedlog", &udl1add) //add
	ESUploader.AddAnyLog_Async("userdefinedlog", &udl2add) //add
	ESUploader.AddAnyLog_Async("userdefinedlog", &udl3add) //add

	time.Sleep(time.Second * 600)

}




```
