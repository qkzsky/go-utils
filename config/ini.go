package config

import (
	"fmt"
	"github.com/qkzsky/go-utils"
	"gopkg.in/ini.v1"
	"log"
	"os"
	"path/filepath"
)

const AppConfFile = "app.ini"

var (
	AppPath string
	AppName string

	defaultConf *ini.File
)

func init() {
	var err error
	if AppPath, err = filepath.Abs(filepath.Dir(os.Args[0])); err != nil {
		panic(err)
	}

	defaultConf = NewConfig(AppConfFile)
	AppName = Key("name").MustString("app")
}

func NewConfig(filename string) *ini.File {
	var configPath string
	configPath = filepath.Join(AppPath, "config", filename)
	if !utils.FileExists(configPath) {
		tempPath, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		configPath = filepath.Join(tempPath, "config", filename)
		for !utils.FileExists(configPath) {
			if tempPath == "" {
				log.Println(fmt.Sprintf("config file %s not existed!", filename))
				return nil
			}
			tempPath = utils.ParentDirectory(tempPath)
			configPath = filepath.Join(tempPath, "config", filename)
		}
	}

	cfg, err := ini.Load(configPath)
	if err != nil {
		panic(err)
	}
	return cfg
}

func Section(name string) *ini.Section {
	return defaultConf.Section(name)
}

func Key(name string) *ini.Key {
	return Section("app").Key(name)
}
