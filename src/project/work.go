package project

import (
	"context"
	"fmt"
	"sync"
	"time"
	"util"
)

func Start(project *Project, group *sync.WaitGroup) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go start(project, group, ctx)
	return cancel
}

func start(project *Project, group *sync.WaitGroup, ctx context.Context) {
	defer group.Done()
	if project == nil {
		return
	}
	util.LogPrint("Start", util.I, "Start",project.ProjectName,fmt.Sprint(project.ProjectName, " start"))
	if err := project.Open(); err != nil {
		util.LogPrint("Start", util.E, "Start",project.ProjectName,fmt.Sprint(project.ProjectName, " start failed, err:", err))
		return
	}
	ch := make(chan []string)
	groupCtl := sync.WaitGroup{}
	groupCtl.Add(3)
	go checkModify(project, ch, ctx, &groupCtl)
	go save(project, ctx, &groupCtl)
	go sftpHandle(project,ch,ctx,&groupCtl)
	groupCtl.Wait()
	if err := project.Write(); err != nil {
		util.LogPrint("Start", util.E, "Start",project.ProjectName,fmt.Sprint(project.ProjectName, " write failed, err:", err))
	}
	project.Close()
	util.LogPrint("Start", util.I, "Start",project.ProjectName,fmt.Sprint(project.ProjectName, " finish"))
}


func checkModify(project *Project, ch chan []string, ctx context.Context, group *sync.WaitGroup) {
	util.LogPrint("modify", util.I, "checkModify",project.ProjectName,fmt.Sprint("checkmodify start"))
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-ctx.Done():
			group.Done()
			util.LogPrint("modify", util.I, "checkModify",project.ProjectName,fmt.Sprint("checkmodify finish"))
			return
		case <-tick:
			util.LogPrint("modify", util.D, "checkModify",project.ProjectName,fmt.Sprint("tick checModify"))
			project.Lock()
			res, err := project.CheckModify()
			project.Unlock()
			if err != nil {
				util.LogPrint("modify", util.E, "checkModify",project.ProjectName,fmt.Sprint("checkmodify failed, err:", err))
			}else{
				if len(res) != 0 {
					util.LogPrint("modify", util.D, "checkModify",project.ProjectName,fmt.Sprint("modify:", res))
					ch <- res
				}
			}

		}
	}
}

func save(project *Project, ctx context.Context, group *sync.WaitGroup) {
	util.LogPrint("save", util.I, "save",project.ProjectName,fmt.Sprint("save start"))
	tick := time.Tick(1 * time.Hour)
	for {
		select {
		case <-ctx.Done():
			group.Done()
			util.LogPrint("save", util.I, "save",project.ProjectName,fmt.Sprint("save finish"))
			return
		case <-tick:
			project.Lock()
			err := project.Write()
			project.Unlock()
			if err != nil {
				util.LogPrint("save", util.E, "save",project.ProjectName,fmt.Sprint("save failed, err:", err))
			}else{
				util.LogPrint("save", util.D, "save",project.ProjectName,fmt.Sprint("save sucess"))
			}
		}
	}
}

func sftpHandle(project *Project, ch chan []string, ctx context.Context, group *sync.WaitGroup) {
	util.LogPrint("sftp", util.I, "sftpHandle",project.ProjectName,fmt.Sprint("sftpHandle start"))
	for {
		select {
		case res, ok := <- ch:
			if ok {
				util.LogPrint("sftp", util.D, "sftpHandle",project.ProjectName,fmt.Sprint("sftpHandle modify:", res))
				project.Lock()
				err := project.Sftp(res)
				project.Unlock()
				if err != nil {
					util.LogPrint("sftp", util.E, "sftpHandle",project.ProjectName,fmt.Sprint("sftpHandle failed, err:", err))
				}
			}
		case <-ctx.Done():
			util.LogPrint("sftp", util.I, "sftpHandle",project.ProjectName,fmt.Sprint("sftpHandle finish"))
			group.Done()
			return
		}
	}
}

