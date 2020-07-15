package dir

import (
	"bytes"
	"sftp"
	"testing"
	"time"
	"fmt"
)

func TestDirectory_Open(t *testing.T) {
	dir := New()
	err := dir.Open(`E:\test_data`)
	if err != nil {
		t.Error("TraversalDir failed, err:", err)
	}
	buffer := make([]byte, 0, 40960)
	buf := bytes.NewBuffer(buffer)


	err = dir.EncodeJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}

	t.Log(buf.String())
}

func TestDirectory_Modify(t *testing.T) {
	dir := New()
	err := dir.Open(`E:\test_data`)
	if err != nil {
		t.Error("TraversalDir failed, err:", err)
	}
	buf := new(bytes.Buffer)
	err = dir.EncodeJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}
	fmt.Println("init:", buf.String())

	address := &sftp.RemoteIpAddress{
		Address: "127.0.0.1:8000",
		User:    "valvrave",
		Passwd:  "valvrave",
		TimeOut: 5 * time.Second,
	}
	client,err := sftp.Dial(address)
	if err != nil {
		t.Errorf("connect failed, err:%v", err)
	}
	defer client.Close()

	if err := dir.Upload(client, "E:\\test_data",
		"/home/valvrave/sftp-test", "\\", "/", []string{"E:\\test_data"}); err != nil {
		t.Errorf("upload failed, err:%v", err)
	}

	buf.Truncate(0)
	err = dir.EncodeJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}
	fmt.Println("same:", buf.String())

	fmt.Println("start sleep........")
	time.Sleep(30 * time.Second)

	modify, err := dir.CheckModify()
	if err != nil {
		t.Error("CheckDirModify failed, err:", err)
	}

	buf.Truncate(0)
	err = dir.EncodeJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}
	fmt.Println("same:", buf.String())

	fmt.Println("modify:", modify)

	fmt.Println("map:", dir.Index)
	if err := dir.Upload(client, "E:\\test_data",
		"/home/valvrave/sftp-test", "\\", "/", modify); err != nil {
		t.Errorf("upload failed, err:%v", err)
	}
	buf.Truncate(0)
	err = dir.EncodeJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}
	fmt.Println("same:", buf.String())
	fmt.Println("map:", dir.Index)
}
