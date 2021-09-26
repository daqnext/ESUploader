# loguploader
server side upload logs to remote elastic search server

## install 
```go
go get github.com/daqnext/logservice
```

## example
```go

package main

import (
	"time"

	"github.com/daqnext/loguploader/uploader"
)

type UserDefinedLog struct {
	Field1 string
	Field2 int
	Ip     string
}

type UserDefinedMapLog struct {
	Field1 string
	Field2 int
	Ip     string
}

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
	logUploader.AddSqldbLog("TestApp9", "sql insert xxxx 5", time.Now().Unix())

	//add taglog
	logUploader.AddTagLog("App1", "tag1", "this is custom message 1", time.Now().Unix())
	logUploader.AddTagLog("App2", "tag2", "this is custom message 2", time.Now().Unix())
	logUploader.AddTagLog("App3", "tag1", "this is custom message 1", time.Now().Unix())

	//userdefined log

	udl1 := UserDefinedLog{"this is filed1", 2, logUploader.GetPublicIP()}
	udl2 := UserDefinedLog{"this is filed2", 2, logUploader.GetPublicIP()}
	udl3 := UserDefinedLog{"this is filed1", 2, logUploader.GetPublicIP()}

	logUploader.AddUserDefinedLog("userdefinedlog", &udl1) //add
	logUploader.AddUserDefinedLog("userdefinedlog", &udl2) //add
	logUploader.AddUserDefinedLog("userdefinedlog", &udl3) //add

	mudl1 := UserDefinedMapLog{"this is filed1", 2, logUploader.GetPublicIP()}
	mudl2 := UserDefinedMapLog{"this is filed2", 2, logUploader.GetPublicIP()}

	logUploader.AddUserDefinedMapLog("userdefinedmaplog", mudl1.Field1, &mudl1)
	logUploader.AddUserDefinedMapLog("userdefinedmaplog", mudl1.Field1, &mudl1) //overwrite
	logUploader.AddUserDefinedMapLog("userdefinedmaplog", mudl2.Field1, &mudl2) //add

	logUploader.Start()

	time.Sleep(time.Second * 600)

}



```
