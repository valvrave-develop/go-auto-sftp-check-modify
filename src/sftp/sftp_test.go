package sftp

import "testing"

func TestSftp__Connect(t *testing.T) {
	client := NewClient("192.168.56.101:22", "valvrave", "valvrave")
	if err := client.Connect(); err != nil {
		t.Errorf("connect failed, err:%v", err)
	}
	client.Close()
}

func TestSftp__Put(t *testing.T) {
	client := NewClient("192.168.56.101:22", "valvrave", "valvrave")
	if err := client.Connect(); err != nil {
		t.Errorf("connect failed, err:%v", err)
	}
	defer client.Close()
	err := client.Put(`E:\workspace\go\auto-scp-check-update\src\sftp\test.txt`, "/home/valvrave/sftp-test/test.txt")
	if err != nil {
		t.Errorf("put failed, err:%v", err)
	}
}

func TestSftp__Mkdir(t *testing.T) {
	client := NewClient("192.168.56.101:22", "valvrave", "valvrave")
	if err := client.Connect(); err != nil {
		t.Errorf("connect failed, err:%v", err)
	}
	defer client.Close()
	err := client.Mkdir("/home/valvrave/sftp-test/mkdir-test")
	if err != nil {
		t.Errorf("put failed, err:%v", err)
	}
}

func TestSftp__Remove(t *testing.T) {
	client := NewClient("192.168.56.101:22", "valvrave", "valvrave")
	if err := client.Connect(); err != nil {
		t.Errorf("connect failed, err:%v", err)
	}
	defer client.Close()
	err := client.Remove( "/home/valvrave/sftp-test/test.txt")
	if err != nil {
		t.Errorf("put failed, err:%v", err)
	}
}

func TestSftp__RemoveDirectory(t *testing.T) {
	client := NewClient("192.168.56.101:22", "valvrave", "valvrave")
	if err := client.Connect(); err != nil {
		t.Errorf("connect failed, err:%v", err)
	}
	defer client.Close()
	err := client.RemoveDirectory("/home/valvrave/sftp-test/mkdir-test")
	if err != nil {
		t.Errorf("put failed, err:%v", err)
	}
}