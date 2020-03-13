package sftp

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"time"
)

type Sftp interface {
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

func Dial(address, user, passwd string, timeout time.Duration) (Sftp, error) {
	s := &sftp_{
		address:address,
		user:user,
		passwd:passwd,
		sshConn:    nil,
		sftpClient: nil,
	}
	callBack := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
	config := &ssh.ClientConfig{
		User: s.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.passwd),
		},
		HostKeyCallback:callBack,
		Timeout:timeout,
	}
	conn, err := ssh.Dial("tcp", s.address, config)
	if err != nil {
		return nil, fmt.Errorf("ssh dial[%s] failed:%v", s.address, err)
	}

	sftp, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create sftp client to [%s] failed:%v", s.address, err)
	}

	s.sshConn = conn
	s.sftpClient = sftp
	return s, nil
}

func (s *sftp_) Close(){
	s.sftpClient.Close()
	s.sshConn.Close()
	s.sshConn = nil
	s.sftpClient = nil
}

//上传文件
func (s *sftp_) Put(local, remote string) error {
	localFp, err := os.Open(local)
	if err != nil {
		return fmt.Errorf("Open %s failed:%v", local, err)
	}
	defer localFp.Close()
	info, err := os.Stat(local)
	if err != nil {
		return fmt.Errorf("Stat %s failed:%v", local, err)
	}
	remoteFp,err := s.sftpClient.OpenFile(remote, os.O_CREATE | os.O_RDWR | os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("sftp OpenFile %s failed:%v", local, err)
	}
	defer remoteFp.Close()
	total := 0
	content := make([]byte, 4096)
	for {
		readLen, err := localFp.Read(content)
		if err != nil && err != io.EOF{
			return fmt.Errorf("local Read %s failed:%v", local, err)
		}
		if _, err := remoteFp.Write(content[:readLen]); err != nil {
			return fmt.Errorf("remote Write %s failed:%v", local, err)
		}
		if err == io.EOF {
			break
		}
		total+=readLen
		if int64(total) >= info.Size() {
			break
		}
	}
	return nil
}
//创建目录，只支持在当前目录创建新目录
func (s *sftp_) Mkdir(remote string) error {
	_, err := s.sftpClient.Stat(remote)
	if err != nil {
		if os.IsNotExist(err) {
			err = s.sftpClient.Mkdir(remote)
			if err != nil {
				return fmt.Errorf("sftp Mkdir %s failed:%v", remote, err)
			}
			return nil
		}
		return fmt.Errorf("sftp Stat %s failed:%v", remote, err)
	}
	return nil
}
//删除文件，不支持删除目录
func (s *sftp_) Remove(remote string) error {
	_, err := s.sftpClient.Stat(remote)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("sftp Stat %s failed:%v", remote, err)
	}
	err = s.sftpClient.Remove(remote)
	if err != nil {
		return fmt.Errorf("sftp Remove %s failed:%v", remote, err)
	}
	return err
}
//删除目录
//API本身不支持包含文件的目录
//此时封装后的api支持删除包含文件的目录
func (s *sftp_) RemoveDirectory(remote string) error {
	_, err := s.sftpClient.Stat(remote)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("sftp Stat %s failed:%v", remote, err)
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
				return fmt.Errorf("sftp Remove %s failed:%v", w.Path(), err)
			}
		}
	}
	for i:=len(dir)-1; i>=0; i-- {
		err = s.sftpClient.RemoveDirectory(dir[i])
		if err != nil {
			return fmt.Errorf("sftp RemoveDirectory %s failed:%v", dir[i], err)
		}
	}
	return nil
}