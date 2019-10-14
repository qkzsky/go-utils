package conf

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qkzsky/go-utils"
	"gopkg.in/ini.v1"
	"log"
	"os"
	"path/filepath"
)

const AppConfFile = "app.ini"

var (
	AppPath string

	AppConf *ini.File
)

func init() {
	var err error
	if AppPath, err = filepath.Abs(filepath.Dir(os.Args[0])); err != nil {
		panic(err)
	}

	AppConf = NewConfig(AppConfFile)
	gin.SetMode(AppConf.Section("").Key("mode").String())
}

func NewConfig(filename string) *ini.File {
	var confPath string
	confPath = filepath.Join(AppPath, "conf", filename)
	if !utils.FileExists(confPath) {
		tempPath, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		confPath = filepath.Join(tempPath, "conf", filename)
		for !utils.FileExists(confPath) {
			if tempPath == "" {
				log.Println(fmt.Sprintf("config file %s not existed!", filename))
				return nil
			}
			tempPath = utils.ParentDirectory(tempPath)
			confPath = filepath.Join(tempPath, "conf", filename)
		}
	}

	cfg, err := ini.Load(confPath)
	if err != nil {
		panic(err)
	}
	return cfg
}
