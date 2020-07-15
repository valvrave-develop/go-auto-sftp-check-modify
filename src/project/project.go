package project

import (
	"conf"
	"context"
	"dir"
	"fmt"
	"os"
	"runtime/debug"
	"sftp"
	"strings"
	"sync"
	"time"
	"util"
)

type Project struct {
	Dirs            *dir.Directory
	config          *conf.ProjectConfig
	remoteAddress   *sftp.RemoteIpAddress
	sftpClient      sftp.Sftp
	localSeparator  string
	remoteSeparator string
	context         context.Context
	cancel          context.CancelFunc
	group           sync.WaitGroup
}

func NewProject(config *conf.ProjectConfig) *Project {
	project := &Project{
		config: config,
		remoteAddress: &sftp.RemoteIpAddress{
			Address: config.RemoteAddress,
			User:    config.User,
			Passwd:  config.Passwd,
			TimeOut: 5 * time.Second,
		},
		sftpClient:      nil,
		Dirs:            dir.New(),
		localSeparator:  separator(config.LocalOs),
		remoteSeparator: separator(config.RemoteOs),
		group:           sync.WaitGroup{},
	}
	project.context, project.cancel = context.WithCancel(context.Background())
	go project.run()
	return project
}

func (p *Project) Close() {
	p.cancel()
	if p.sftpClient != nil {
		p.sftpClient.Close()
	}
	p.group.Wait()
}

func (p *Project) init() {
	cli, err := sftp.Dial(p.remoteAddress)
	if err != nil {
		panic(err)
	}
	p.sftpClient = cli

	if err := p.read(); err != nil {
		panic(err)
	}
}

func (p *Project) run() {
	p.init()
	checkTicker := time.NewTicker(2 * time.Second)
	saveTicker := time.NewTicker(10 * time.Minute)
	p.group.Add(1)
	defer func() {
		checkTicker.Stop()
		saveTicker.Stop()
		if err := p.write(); err != nil {
			util.LogPrint("project", util.E, "Close", p.config.Name, fmt.Sprint("write failed:", err))
		}
		p.group.Done()
	}()
	for {
		select {
		case <-p.context.Done():
			util.LogPrint("project", util.I, "Run", p.config.Name, "watch finish")
			return
		case <-checkTicker.C:
			func() {
				defer func() {
					if err := recover(); err != nil {
						util.LogPrint("project", util.E, "Panic", p.config.Name, fmt.Sprint(err, "\n", string(debug.Stack())))
					}
				}()
				if res, err := p.Dirs.CheckModify(); err != nil {
					util.LogPrint("project", util.E, "Run", p.config.Name, fmt.Sprint("check modify failed:", err))
				} else {
					if len(res) > 0 {
						util.LogPrint("project", util.I, "Run", p.config.Name, fmt.Sprint("upload start:", res))
						if err := p.sftp(res); err != nil {
							util.LogPrint("project", util.E, "Run", p.config.Name, fmt.Sprint("sftp upload failed:", err))
						} else {
							util.LogPrint("project", util.I, "Run", p.config.Name, "upload finish")
						}
					}
				}
			}()
		case <-saveTicker.C:
			if err := p.write(); err != nil {
				util.LogPrint("project", util.E, "Run", p.config.Name, fmt.Sprint("save file failed:", err))
			}
		}
	}
}

func (p *Project) write() error {
	temp := fmt.Sprint(p.config.SaveProject, ".temp")
	err := func() error {
		fp, err := os.Create(temp)
		if err != nil {
			return err
		}
		defer fp.Close()
		if err := p.Dirs.EncodeJson(fp); err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		os.Remove(temp)
		return err
	}
	return os.Rename(temp, p.config.SaveProject)
}
func (p *Project) read() error {
	fp, err := os.Open(p.config.SaveProject)
	if err != nil {
		if os.IsNotExist(err) {
			err = p.Dirs.Open(p.config.LocalBaseDir)
			if err != nil {
				return err
			}
			return p.Dirs.Upload(p.sftpClient, p.config.LocalBaseDir, p.config.RemoteBaseDir, p.localSeparator, p.remoteSeparator, []string{p.Dirs.Dir.Name})
		}
		return err
	}
	defer fp.Close()
	return p.Dirs.DecodeJson(fp)
}

func (p *Project) sftp(modify []string) error {
	err := p.Dirs.Upload(p.sftpClient, p.config.LocalBaseDir, p.config.RemoteBaseDir, p.localSeparator, p.remoteSeparator, modify)
	for i := 1; i >= 0; i-- {
		if err == nil {
			return err
		}
		cli, err := sftp.Dial(p.remoteAddress)
		if err != nil {
			return err
		}
		p.sftpClient.Close()
		p.sftpClient = cli
		err = p.Dirs.Upload(p.sftpClient, p.config.LocalBaseDir, p.config.RemoteBaseDir, p.localSeparator, p.remoteSeparator, modify)
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
