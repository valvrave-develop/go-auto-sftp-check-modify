package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	Open = true
	E = "ERROR"
	I = "INFO"
	D = "DEBUG"
	Sync = false
	Level = D
	LogFile = `E:\workspace\go\go-auto-sftp-check-modify\src\log\error.log`
)

var RecordLog *os.File
var chanLog chan string

func init() {
	if !Open && Sync {
		return
	}
	recordLogFile := fmt.Sprint(LogFile,".",time.Now().Format("20060102-150405"))
	fp, err := os.Create(recordLogFile)
	if err != nil {
		panic(fmt.Sprintf("create log file failed, fileName:%s", LogFile))
	}
	RecordLog = fp
	chanLog = make(chan string, 100)
	go func(){
		for  {
			select {
			case res := <-chanLog:
				RecordLog.WriteString(res)
			}
		}
	}()
}

var LogPrint func(string, string, string, string, string) = func(desc string, level string, funcName string, label string, msg string){
	if !Open {
		return
	}
	if Level == I && level == D {
		return
	}
	if Level == E && (level == D || level == I) {
		return
	}
	now := time.Now().Format("2006-1-2 15:04:05.000000000")
	_, file, n, ok := runtime.Caller(1)
	var info string = ""
	if ok {
		info = fmt.Sprintf("[%s][%s][%s][%s:%d][%s][%s]%s\n", now, desc, level, filepath.Base(file), n, funcName, label, msg )
	}else{
		info = fmt.Sprintf("[%s][%s][%s][%s][%s]%s\n", now, desc, level, funcName, label, msg )
	}
	if Sync {
		fmt.Print(info)
		return
	}
	chanLog <- info
}
