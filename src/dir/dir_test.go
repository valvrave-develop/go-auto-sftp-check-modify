package dir

import (
	"bytes"
	"fmt"
	"sftp"
	"testing"
	"time"
)

func TestDirectory_Open(t *testing.T) {
	dir := New()
	err := dir.Open(`E:\test_data`)
	if err != nil {
		t.Error("TraversalDir failed, err:", err)
	}
	buffer := make([]byte, 0, 40960)
	buf := bytes.NewBuffer(buffer)


	err = dir.WriteJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}

	t.Log(buf.String())
}

func TestDir_CheckDirModify(t *testing.T) {
	dir := New()
	err := dir.Open(`E:\test_data`)
	if err != nil {
		t.Error("TraversalDir failed, err:", err)
	}
	buf := new(bytes.Buffer)
	err = dir.WriteJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}
	fmt.Println("init:", buf.String())
	buf.Truncate(0)

	time.Sleep(5 * time.Second)

	modify, err := dir.CheckModify()
	if err != nil {
		fmt.Println("check modify failed, err:", err)
	}else{
		fmt.Println(modify)
	}

}

func TestDirectory_Modify(t *testing.T) {
	dir := New()
	err := dir.Open(`E:\test_data`)
	if err != nil {
		t.Error("TraversalDir failed, err:", err)
	}
	buf := new(bytes.Buffer)
	err = dir.WriteJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}
	fmt.Println("init:", buf.String())

	client := sftp.NewClient("192.168.56.101:22", "valvrave", "valvrave")
	if err := client.Connect(); err != nil {
		t.Errorf("connect failed, err:%v", err)
	}
	defer client.Close()

	if err := dir.Modify(client, "E:\\test_data",
		"/home/valvrave/sftp-test", "\\", "/", []string{"E:\\test_data"}); err != nil {
		t.Errorf("upload failed, err:%v", err)
	}

	buf.Truncate(0)
	err = dir.WriteJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}
	fmt.Println("same:", buf.String())

	fmt.Println("start sleep........")
	time.Sleep(20 * time.Second)

	modify, err := dir.CheckModify()
	if err != nil {
		t.Error("CheckDirModify failed, err:", err)
	}

	buf.Truncate(0)
	err = dir.WriteJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}
	fmt.Println("same:", buf.String())

	fmt.Println("modify:", modify)

	fmt.Println("map:", dir.(*directory).dirIndex)
	if err := dir.Modify(client, "E:\\test_data",
		"/home/valvrave/sftp-test", "\\", "/", modify); err != nil {
		t.Errorf("upload failed, err:%v", err)
	}
	buf.Truncate(0)
	err = dir.WriteJson(buf)
	if err != nil {
		t.Error("write failed, err:",err)
	}
	fmt.Println("same:", buf.String())
	fmt.Println("map:", dir.(*directory).dirIndex)
}
