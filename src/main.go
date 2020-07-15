package main

import (
	"conf"
	"fmt"
	"log"
	"os"
	"os/signal"
	"project"
	"syscall"
	"web"
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
	projects := make(map[string]*project.Project)
	for _, conf := range config.Conf {
		if conf.Switch != "on" {
			continue
		}
		projects[conf.Name] = project.NewProject(conf)
	}
	go web.WebServerStart(projects)
	wait()
	for _, p := range projects {
		p.Close()
	}
	fmt.Println("auto-upload-file finish")
}
