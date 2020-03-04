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
	err := client.Put(`E:\workspace\go\auto-scp-check-update\src\sftp\test.txt`, "/home/valvrave/sftp-test")
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
	err := client.Mkdir("mkdir-test", "/home/valvrave/sftp-test")
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
	err := client.Remove("test.txt", "/home/valvrave/sftp-test")
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
	err := client.RemoveDirectory("mkdir-test", "/home/valvrave/sftp-test")
	if err != nil {
		t.Errorf("put failed, err:%v", err)
	}
}