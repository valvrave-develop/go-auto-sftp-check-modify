package project

import (
	"bytes"
	"conf"
	"encoding/json"
	"testing"
)

func TestProject_Open(t *testing.T) {
	configure := `
{
  "project": [{
    "switch": "on",
    "name": "project_test",
    "local_os": "Windows",
    "local_base_dir": "E:\\workspace\\go\\go-auto-sftp-check-modify\\src\\project",
    "remote_os": "Linux",
    "remote_base_dir": "/home/valvrave/sftp-test",
    "remote_address": "192.168.56.101:22",
    "user": "valvrave",
    "passwd": "valvrave",
    "save_project": "E:\\workspace\\go\\go-auto-sftp-check-modify\\src\\project\\save.txt"
  }
  ]
}
`
	config := new(conf.Config)
	decode := json.NewDecoder(bytes.NewBufferString(configure))
	err := decode.Decode(config)
	if err != nil {
		t.Error(err)
	}
	t.Log(config.Conf)
	p, err := Open(config.Conf[0])
	if err != nil {
		t.Error(err)
	}
	p.Close()
}