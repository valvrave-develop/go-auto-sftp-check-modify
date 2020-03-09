package sftp

import (
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"time"
	"util"
)

type Sftp interface {
	Connect() error
	Close()
	Put(local, remote string) error
	Mkdir(remote string) error
	Remove(remote string) error
	RemoveDirectory(remote string) error
}


type  sftp_ struct{
	address string
	user string
	passwd string
	sshConn *ssh.Client
	sftpClient *sftp.Client
}

func NewClient(address, user, passwd string) Sftp {
	return &sftp_{
		address:address,
		user:user,
		passwd:passwd,
		sshConn:    nil,
		sftpClient: nil,
	}
}

func (s *sftp_) Connect() error {
	callBack := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
	config := &ssh.ClientConfig{
		User: s.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.passwd),
		},
		HostKeyCallback:callBack,
		Timeout:5*time.Second,
	}
	conn, err := ssh.Dial("tcp", s.address, config)
	if err != nil {
		return fmt.Errorf("ssh connect[%s] failed, errMsg:%v", s.address, err)
	}

	sftp, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("create sftp client to [%s] failed, errMsg:%v", s.address, err)
	}

	s.sshConn = conn
	s.sftpClient = sftp
	return nil
}

func (s *sftp_) Close(){
	s.sftpClient.Close()
	s.sshConn.Close()
	s.sshConn = nil
	s.sftpClient = nil
}

//上传文件
func (s *sftp_) Put(local, remote string) error {
	util.LogPrint("sftp", util.I, "Put", local, fmt.Sprint(remote, " Put start"))
	var e error
	defer func(){
		if e == nil {
			util.LogPrint("sftp", util.I, "Put", local, fmt.Sprint(remote, " Put sucess"))
		}else{
			util.LogPrint("sftp", util.E, "Put", local, fmt.Sprint(remote, " Put errMsg:", e))
		}
	}()
	localFp, err := os.Open(local)
	if err != nil {
		e = errors.New(fmt.Sprintf("Open %s failed:%v", local, err))
		return e
	}
	defer localFp.Close()
	info, err := os.Stat(local)
	if err != nil {
		e = errors.New(fmt.Sprintf("Stat %s failed:%v", local, err))
		return e
	}
	remoteFp,err := s.sftpClient.OpenFile(remote, os.O_CREATE | os.O_RDWR | os.O_TRUNC)
	if err != nil {
		e = errors.New(fmt.Sprintf("sftp OpenFile %s failed:%v", local, err))
		return e
	}
	defer remoteFp.Close()
	total := 0
	content := make([]byte, 4096)
	for {
		readLen, err := localFp.Read(content)
		if err != nil && err != io.EOF{
			e = errors.New(fmt.Sprintf("local Read %s failed:%v", local, err))
			return e
		}
		if _, err := remoteFp.Write(content[:readLen]); err != nil {
			e = errors.New(fmt.Sprintf("remote Write %s failed:%v", local, err))
			return e
		}
		if err == io.EOF {
			break
		}
		total+=readLen
		if int64(total) >= info.Size() {
			break
		}
	}
	e = nil
	return e
}
//创建目录，只支持在当前目录创建新目录
func (s *sftp_) Mkdir(remote string) error {
	util.LogPrint("sftp", util.I, "Mkdir", remote, "Mkdir start")
	_, err := s.sftpClient.Stat(remote)
	defer func(){
		if err == nil {
			util.LogPrint("sftp", util.I, "Mkdir", remote, "Mkdir sucess")
		}else{
			util.LogPrint("sftp", util.E, "Mkdir", remote, fmt.Sprint("Mkdir failed:", err))
		}
	}()
	if err != nil {
		if os.IsNotExist(err) {
			err = s.sftpClient.Mkdir(remote)
			return err
		}
		return err
	}
	return nil
}
//删除文件，不支持删除目录
func (s *sftp_) Remove(remote string) error {
	util.LogPrint("sftp", util.I, "Remove", remote, "Remove start")
	_, err := s.sftpClient.Stat(remote)
	defer func(){
		if err == nil {
			util.LogPrint("sftp", util.I, "Remove", remote, "Remove sucess")
		}else{
			util.LogPrint("sftp", util.E, "Remove", remote, fmt.Sprint("Remove failed:", err))
		}
	}()
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			return err
		}
		return err
	}
	err = s.sftpClient.Remove(remote)
	return err
}
//删除目录
//API本身不支持包含文件的目录
//此时封装后的api支持删除包含文件的目录
func (s *sftp_) RemoveDirectory(remote string) error {
	util.LogPrint("sftp", util.I, "RemoveDirectory", remote, "RemoveDirectory start")
	_, err := s.sftpClient.Stat(remote)
	defer func(){
		if err == nil {
			util.LogPrint("sftp", util.I, "RemoveDirectory", remote, "RemoveDirectory sucess")
		}else{
			util.LogPrint("sftp", util.E, "RemoveDirectory", remote, fmt.Sprint("RemoveDirectory failed:", err))
		}
	}()
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			return err
		}
		err = errors.New(fmt.Sprintf("sftp Stat %s failed:%v", remote, err))
		return err
	}
	w := s.sftpClient.Walk(remote)
	dir := make([]string, 0, 10)
	for w.Step() {
		if w.Err() != nil {
			continue
		}
		if w.Stat().IsDir(){
			dir = append(dir, w.Path())
		}else{
			err = s.sftpClient.Remove(w.Path())
			if err != nil {
				err = errors.New(fmt.Sprintf("sftp Remove %s failed:%v", w.Path(), err))
				return err
			}
		}
	}
	for i:=len(dir)-1; i>=0; i-- {
		err = s.sftpClient.RemoveDirectory(dir[i])
		if err != nil {
			err = errors.New(fmt.Sprintf("sftp RemoveDirectory %s failed:%v", dir[i], err))
			return err
		}
	}
	return nil
}