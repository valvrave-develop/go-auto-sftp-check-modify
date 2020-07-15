package sftp

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"time"
	"util"
)

type RemoteIpAddress struct {
	Address string
	User    string
	Passwd  string
	TimeOut time.Duration
}

type Sftp interface {
	Close()
	Put(local, remote string) error
	Mkdir(remote string) error
	Remove(remote string) error
	RemoveDirectory(remote string) error
}

type sftp_ struct {
	remote     *RemoteIpAddress
	sshConn    *ssh.Client
	sftpClient *sftp.Client
}

func Dial(address *RemoteIpAddress) (Sftp, error) {
	s := &sftp_{
		remote:     address,
		sshConn:    nil,
		sftpClient: nil,
	}
	callBack := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
	config := &ssh.ClientConfig{
		User: s.remote.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.remote.Passwd),
		},
		HostKeyCallback: callBack,
		Timeout:         s.remote.TimeOut,
	}
	conn, err := ssh.Dial("tcp", s.remote.Address, config)
	if err != nil {
		return nil, fmt.Errorf("ssh dial[%s] failed:%v", s.remote.Address, err)
	}

	sftp, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create sftp client to [%s] failed:%v", s.remote.Address, err)
	}

	s.sshConn = conn
	s.sftpClient = sftp
	return s, nil
}

func (s *sftp_) Close() {
	s.sftpClient.Close()
	s.sshConn.Close()
	s.sshConn = nil
	s.sftpClient = nil
}

func (s *sftp_) Put(local, remote string) error {
	defer util.LogPrint("sftp", util.D, "Put", "upload file", fmt.Sprintf("local:%s remote:%s", local, remote))
	return s.put(local, remote)
}
func (s *sftp_) Mkdir(remote string) error {
	defer util.LogPrint("sftp", util.D, "Mkdir", "create director", fmt.Sprintf("remote:%s", remote))
	return s.mkdir(remote)
}
func (s *sftp_) Remove(remote string) error {
	defer util.LogPrint("sftp", util.D, "Remove", "remove file", fmt.Sprintf("remote:%s", remote))
	return s.remove(remote)
}
func (s *sftp_) RemoveDirectory(remote string) error {
	defer util.LogPrint("sftp", util.D, "RemoveDirectory", "remove dircetory", fmt.Sprintf("remote:%s", remote))
	return s.removeDirectory(remote)
}

//上传文件
func (s *sftp_) put(local, remote string) error {
	localFp, err := os.Open(local)
	if err != nil {
		return err
	}
	defer localFp.Close()
	remoteFp, err := s.sftpClient.OpenFile(remote, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer remoteFp.Close()
	content := make([]byte, 4096)
	for {
		readLen, err := localFp.Read(content)
		if err != nil && err != io.EOF {
			return err
		}
		if _, err := remoteFp.Write(content[:readLen]); err != nil {
			return err
		}
		if err == io.EOF {
			break
		}
	}
	return nil
}

//创建目录，只支持在当前目录创建新目录
func (s *sftp_) mkdir(remote string) error {
	_, err := s.sftpClient.Stat(remote)
	if err != nil {
		if os.IsNotExist(err) {
			return s.sftpClient.Mkdir(remote)
		}
	}
	return err
}

//删除文件，不支持删除目录
func (s *sftp_) remove(remote string) error {
	_, err := s.sftpClient.Stat(remote)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return s.sftpClient.Remove(remote)
}

//删除目录
//API本身不支持包含文件的目录
//此时封装后的api支持删除包含文件的目录
func (s *sftp_) removeDirectory(remote string) error {
	_, err := s.sftpClient.Stat(remote)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	w := s.sftpClient.Walk(remote)
	dir := make([]string, 0, 10)
	for w.Step() {
		if w.Err() != nil {
			continue
		}
		if w.Stat().IsDir() {
			dir = append(dir, w.Path())
		} else {
			err = s.sftpClient.Remove(w.Path())
			if err != nil {
				return err
			}
		}
	}
	for i := len(dir) - 1; i >= 0; i-- {
		err = s.sftpClient.RemoveDirectory(dir[i])
		if err != nil {
			return err
		}
	}
	return nil
}
