package conf

import (
	"os"
	"testing"
)

func TestInitConfig(t *testing.T) {
	os.Args[1] = `..\etc\project.json`
	config, err := InitConfig()
	if err != nil {
		t.Error(err)
	}
	for _, v := range config.Conf {
		t.Logf("%v\n", *v)
	}
}
