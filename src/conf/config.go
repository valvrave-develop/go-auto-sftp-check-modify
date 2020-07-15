package conf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Conf []*ProjectConfig `json:"project"`
}

type ProjectConfig struct {
	Switch string           `json:"switch"`
	Name string             `json:"name"`
	User string             `json:"user"`
	Passwd string           `json:"passwd"`
	LocalBaseDir string     `json:"local_base_dir"`
	RemoteAddress string    `json:"remote_address"`
	RemoteBaseDir string    `json:"remote_base_dir"`
	SaveProject   string    `json:"save_project"`
	LocalOs  string         `json:"local_os"`
	RemoteOs string         `json:"remote_os"`
}


func InitConfig() (*Config, error) {
	config := new(Config)
	configFile := fmt.Sprintf("%s%setc%sproject.json", filepath.Dir(os.Args[0]), string(filepath.Separator), string(filepath.Separator))
	if len(os.Args) >= 2 && len(os.Args[1]) != 0 {
		configFile = os.Args[1]
	}
	fp, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	decode := json.NewDecoder(fp)
	err = decode.Decode(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}