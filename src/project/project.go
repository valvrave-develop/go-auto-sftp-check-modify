package project

import (
	"conf"
	"dir"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sftp"
	"strings"
	"sync"
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
	localSeparator string
	remoteSeparator string
	dirFp *os.File
	fp *os.File
	dirs dir.Directory
	client sftp.Sftp
	mutex sync.Mutex
}

func NewProject(conf *conf.ConfigElement) *Project {
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
		dirs:dir.New(),
		mutex:sync.Mutex{},
	}
	return project
}

func (p *Project) Open() error {
	if !filepath.IsAbs(p.LocalBaseDir) {
		return fmt.Errorf("%s is not absolute path", p.LocalBaseDir)
	}
	//锁住当前基目录，防止被手动删除
	if dirFp, err := os.OpenFile(p.LocalBaseDir, os.O_RDONLY, os.ModeDir); err != nil {
		return err
	}else{
		p.dirFp = dirFp
	}
	cli := sftp.NewClient(p.RemoteAddress, p.User, p.Passwd)
	if err := cli.Connect(); err != nil{
		return  err
	}
	p.client = cli
	_, err := os.Stat(p.SaveProject)
	if err != nil {
		if os.IsNotExist(err) {
			p.fp, err = os.Create(p.SaveProject)
			if err != nil {
				return err
			}
			err = p.dirs.Open(p.LocalBaseDir) //遍历
			if err != nil {
				p.fp.Close()
				return err
			}
			initModify := make([]string,1)
			initModify[0] = p.LocalBaseDir
			err = p.Sftp(initModify)               //上传
			if err != nil {
				p.fp.Close()
				return err
			}
			if err := p.Write(); err != nil {    //保存
				p.fp.Close()
				return err
			}
		}else {
			return err
		}
	}else{
		p.fp, err = os.OpenFile(p.SaveProject, os.O_RDWR, 0666)
		if err != nil {
			return err
		}
		if err := p.Read(); err != nil {
			p.fp.Close()
			return err
		}
	}
	return nil
}
func (p *Project) Close(){
	p.dirFp.Close()
	p.dirFp = nil
	p.fp.Close()
	p.fp = nil
	p.dirs = nil
	p.client.Close()
}

func (p *Project) Write() error {
	p.fp.Truncate(0)
	p.fp.Seek(0, io.SeekStart)
	if err := p.dirs.WriteJson(p.fp); err != nil {
		return err
	}
	if err := p.fp.Sync(); err != nil {
		return err
	}
	return nil
}
func (p *Project) Read() error {
	return p.dirs.ReadJson(p.fp)
}

func (p *Project) CheckModify() ([]string, error) {
	return p.dirs.CheckModify()
}

func (p *Project) Sftp( modify []string) error {
	err := p.dirs.Modify(p.client, p.LocalBaseDir, p.RemoteBaseDir, p.localSeparator, p.remoteSeparator, modify)
	for i:=1; i>=0; i-- {
		if err == nil {
			return err
		}
		p.client.Close()
		cli := sftp.NewClient(p.RemoteAddress, p.User, p.Passwd)
		if err := cli.Connect(); err != nil{
			return  err
		}
		p.client = cli
		err = p.dirs.Modify(p.client, p.LocalBaseDir, p.RemoteBaseDir, p.localSeparator, p.remoteSeparator, modify)
	}
	return err
}

func (p *Project) Lock(){
	p.mutex.Lock()
}
func (p *Project) Unlock(){
	p.mutex.Unlock()
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