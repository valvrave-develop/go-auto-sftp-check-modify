package sftp

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
	"util"
)

type Sftp interface {
	Connect() error
	Close()
	Put(file, remotePath string) error
	Mkdir(path, remotePath string) error
	Remove(file, remotePath string) error
	RemoveDirectory(path, remotePath string) error
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
		return fmt.Errorf("ssh connect failed, errMsg:%v", err)
	}

	sftp, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("create sftp client failed, errMsg:%v", err)
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

func (s *sftp_) Put(file, remotePath string) error {
	absFile := fmt.Sprintf("%s/%s",remotePath,filepath.Base(file))
	util.LogPrint("sftp", util.I, "Put", file, fmt.Sprint(absFile, " put start"))
	localFp, err := os.Open(file)
	if err != nil {
		return err
	}
	defer localFp.Close()
	info, err := os.Stat(file)
	if err != nil {
		return err
	}
	_, err = s.sftpClient.Stat(absFile)
	exist := true
	if err != nil {
		if os.IsNotExist(err) {
			exist = false
		}else{
			return err
		}
	}
	var remoteFp *sftp.File = nil
	if exist {
		remoteFp,err = s.sftpClient.OpenFile(absFile, os.O_RDWR | os.O_TRUNC)
	}else{
		remoteFp,err = s.sftpClient.Create(absFile)
	}
	if err != nil {
		return err
	}
	defer remoteFp.Close()
	total := 0
	content := make([]byte, 4096)
	for {
		readLen, err := localFp.Read(content)
		if err != nil && err != io.EOF{
			return err
		}
		if _, err := remoteFp.Write(content[:readLen]); err != nil {
			return err
		}
		if err == io.EOF {
			break
		}
		total+=readLen
		if int64(total) >= info.Size() {
			break
		}
	}
	util.LogPrint("sftp", util.I, "Put", file, fmt.Sprint(absFile, " put sucess"))
	return nil
}

func (s *sftp_) Mkdir(path, remotePath string) error {
	absPath := fmt.Sprintf("%s/%s",remotePath,filepath.Base(path))
	util.LogPrint("sftp", util.I, "Mkdir", path, fmt.Sprint(absPath, " mkdir start"))
	_, err := s.sftpClient.Stat(absPath)
	defer func(){
		if err == nil {
			util.LogPrint("sftp", util.I, "Mkdir", path, fmt.Sprint(absPath, " mkdir sucess"))
		}
	}()
	if err != nil {
		if os.IsNotExist(err) {
			err = s.sftpClient.Mkdir(absPath)
			return err
		}
		return err
	}
	return nil
}

func (s *sftp_) Remove(file, remotePath string) error {
	absFile := fmt.Sprintf("%s/%s",remotePath,filepath.Base(file))
	util.LogPrint("sftp", util.I, "Remove", file, fmt.Sprint(absFile, " Remove start"))
	_, err := s.sftpClient.Stat(absFile)
	defer func(){
		if err == nil {
			util.LogPrint("sftp", util.I, "Remove", file, fmt.Sprint(absFile, " Remove sucess"))
		}
	}()
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			return err
		}
		return err
	}
	err = s.sftpClient.Remove(absFile)
	return err
}

func (s *sftp_) RemoveDirectory(path, remotePath string) error {
	absPath := fmt.Sprintf("%s/%s",remotePath,filepath.Base(path))
	util.LogPrint("sftp", util.I, "RemoveDirectory", path, fmt.Sprint(absPath, " RemoveDirectory start"))
	_, err := s.sftpClient.Stat(absPath)
	defer func(){
		if err == nil {
			util.LogPrint("sftp", util.I, "RemoveDirectory", path, fmt.Sprint(absPath, " RemoveDirectory sucess"))
		}
	}()
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			return err
		}
		return err
	}
	w := s.sftpClient.Walk(absPath)
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
				return err
			}
		}
	}
	for i:=len(dir)-1; i>=0; i-- {
		err = s.sftpClient.RemoveDirectory(dir[i])
		if err != nil {
			return err
		}
	}
	return nil
}