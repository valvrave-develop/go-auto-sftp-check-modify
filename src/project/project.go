package project

import (
	"conf"
	"context"
	"dir"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sftp"
	"strings"
	"sync"
	"time"
	"util"
)

const (
	sftpTimeout = 5 * time.Second
)

type Project struct {
	ProjectName string
	User string
	Passwd string
	LocalBaseDir string
	LocalSeparator string
	RemoteAddress string
	RemoteBaseDir string
	RemoteSeparator string
	SaveProject string
	Dirs *dir.Directory

	localSeparator string
	remoteSeparator string
	dirFp *os.File
	fp *os.File
	client sftp.Sftp
	ctx context.Context
	cancel context.CancelFunc
	group sync.WaitGroup
}

func newProject(conf *conf.ProjectConfig) *Project {
	project := &Project{
		ProjectName:conf.Name,
		User:conf.User,
		Passwd:conf.Passwd,
		LocalBaseDir:conf.LocalBaseDir,
		RemoteAddress:conf.RemoteAddress,
		RemoteBaseDir:conf.RemoteBaseDir,
		SaveProject:conf.SaveProject,
		localSeparator:separator(conf.LocalOs),
		remoteSeparator:separator(conf.RemoteOs),
		dirFp:nil,
		fp:nil,
		Dirs:dir.New(),
		group:sync.WaitGroup{},
	}
	project.ctx, project.cancel = context.WithCancel(context.Background())
	return project
}

func Open(conf *conf.ProjectConfig) (*Project, error) {
	p := newProject(conf)
	if !filepath.IsAbs(p.LocalBaseDir) {
		return nil, fmt.Errorf("%s is not absolute path", p.LocalBaseDir)
	}
	//锁住当前基目录，防止被手动删除
	if dirFp, err := os.OpenFile(p.LocalBaseDir, os.O_RDONLY, os.ModeDir); err != nil {
		return nil, err
	}else{
		p.dirFp = dirFp
	}
	cli,err := sftp.Dial(p.RemoteAddress, p.User, p.Passwd, sftpTimeout)
	if err != nil{
		return  nil, err
	}
	p.client = cli
	p.fp, err = os.OpenFile(p.SaveProject, os.O_RDWR, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			if p.fp, err = os.Create(p.SaveProject); err != nil {
				return nil, err
			}
			defer func(){
				if err != nil {
					p.fp.Close()
				}
			}()
			if err = p.Dirs.Open(p.LocalBaseDir); err != nil { //遍历
				return nil, err
			}
			if err = p.sftp([]string{p.LocalBaseDir}); err != nil {          //上传
				return nil,  err
			}
			if err := p.write(); err != nil {    //保存
				return nil, err
			}
		}else {
			return nil, err
		}
	}else{
		if err := p.read(); err != nil {
			p.fp.Close()
			return nil, err
		}
	}
	go p.run()
	return p, nil
}
func (p *Project) Close(){
	p.cancel()
	p.group.Wait()
	p.write()
	p.dirFp.Close()
	p.fp.Close()
	p.client.Close()
}


func (p *Project) run(){
	run := func(){
		p.group.Add(1)
		checkTimer := time.NewTicker(2 * time.Second)
		saveTimer := time.NewTicker(30 * time.Minute)
		modifyCh := make(chan []string)
		defer func(){
			checkTimer.Stop()
			saveTimer.Stop()
			close(modifyCh)
			p.group.Done()
		}()
		var e error
		for {
			e = nil
			util.LogPrint("project", util.I, "Run",p.ProjectName,"watch")
			select {
			case <-p.ctx.Done():
				util.LogPrint("project", util.I, "Run",p.ProjectName,"watch finish")
				return
			case <-checkTimer.C:
				if res, err := p.Dirs.CheckModify(); err != nil {
					e = fmt.Errorf("check failed:", err)
				}else{
					if len(res) >0 {
						util.LogPrint("project", util.I, "Run",p.ProjectName, fmt.Sprint("modify:", res))
						go func(){
							defer func(){
								recover()
							}()
							modifyCh <- res
						}()
					}
				}
			case <-saveTimer.C:
				if err := p.write(); err != nil {
					e = fmt.Errorf("save failed:", err)
				}
			case res, ok := <-modifyCh:
				if ok {
					if err := p.sftp(res); err != nil {
						e = fmt.Errorf("upload failed:", err)
					}
					util.LogPrint("project", util.I, "Run",p.ProjectName, fmt.Sprint("upload finish:", res))
				}
			}
			if e != nil {
				util.LogPrint("project", util.I, "Run",p.ProjectName,fmt.Sprint("watch failed:", e))
			}
		}
	}
	go run()
}

func (p *Project) write() error {
	p.fp.Truncate(0)
	p.fp.Seek(0, io.SeekStart)
	if err := p.Dirs.EncodeJson(p.fp); err != nil {
		return err
	}
	if err := p.fp.Sync(); err != nil {
		return err
	}
	return nil
}
func (p *Project) read() error {
	return p.Dirs.DecodeJson(p.fp)
}

func (p *Project) sftp( modify []string) error {
	err := p.Dirs.Upload(p.client, p.LocalBaseDir, p.RemoteBaseDir, p.localSeparator, p.remoteSeparator, modify)
	for i:=1; i>=0; i-- {
		if err == nil {
			return err
		}
		cli,err := sftp.Dial(p.RemoteAddress, p.User, p.Passwd, sftpTimeout)
		if err != nil{
			return  err
		}
		p.client.Close()
		p.client = cli
		err = p.Dirs.Upload(p.client, p.LocalBaseDir, p.RemoteBaseDir, p.localSeparator, p.remoteSeparator, modify)
	}
	return err
}


func separator(os string) string {
	osUpper := strings.ToUpper(os)
	switch osUpper {
	case "WINDOWS":
		return "\\"
	case "LINUX":
		fallthrough
	case "MAC":
		return "/"
	}
	panic(fmt.Sprint("unknown os type:", os))
}