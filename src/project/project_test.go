package project

import (
	"conf"
	"os"
	"testing"
)

func TestProject_Open(t *testing.T) {
	os.Args[1] = "..\\etc\\project.json"
	config,err := conf.InitConfig()
	if err != nil {
		t.Error("init configure failed, errMsg:", err)
	}
	t.Log(config.Conf)
	config.Conf[0].SaveProject="..\\save\\save.txt"
	p := NewProject(config.Conf[0])
	err = p.Open()
	if err != nil {
		t.Error(err)
	}

	t.Log(p.dirs.DirMap)

	p.Close()
}