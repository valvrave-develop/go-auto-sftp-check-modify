package main

import (
	"conf"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"project"
	"sync"
	"syscall"
	"util"
)

func wait(){
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGQUIT, syscall.SIGKILL,syscall.SIGABRT, syscall.SIGTERM,syscall.SIGINT)
	<- ch
}


func main(){

	config,err := conf.InitConfig()
	if err != nil {
		log.Fatalln("init configure failed, errMsg:", err)
	}
	group := sync.WaitGroup{}
	projects := make(map[string]context.CancelFunc)
	for _, conf := range config.Conf {
		if conf.Switch != "on" {
			continue
		}
		group.Add(1)
		projects[conf.Name] = project.Start(project.NewProject(conf), &group)
	}
	wait()
	for _, cancel := range projects {
		cancel()
	}
	group.Wait()
	util.LogPrint("main", util.I, "main","auto-upload-file",fmt.Sprint("auto-upload-file finish"))
}
