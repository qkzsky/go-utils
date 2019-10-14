package fds

import (
	"apollo_cron/utils/conf"
	"fmt"
	"github.com/XiaoMi/galaxy-fds-sdk-golang"
	"github.com/XiaoMi/galaxy-fds-sdk-golang/Model"
	"gopkg.in/ini.v1"
	"sync"
)

var (
	fdsConf *ini.Section
	fdsMap  sync.Map
	mu      sync.Mutex
)

type FDSClient struct {
	bucketName string
	*galaxy_fds_sdk_golang.FDSClient
}

func (c *FDSClient) GetBucketName() string {
	return c.bucketName
}

func (c *FDSClient) PutObject(objectName string, data []byte, contentType string, headers *map[string]string) (*Model.PutObjectResult, error) {
	return c.Put_Object(c.bucketName, objectName, data, contentType, headers)
}

func (c *FDSClient) SetPublic(objectName string, disablePrefetch bool) (bool, error) {
	return c.Set_Public(c.bucketName, objectName, disablePrefetch)
}

func (c *FDSClient) RefreshObject(objectName string) (bool, error) {
	return c.Refresh_Object(c.bucketName, objectName)
}

func (c *FDSClient) GenerateDownloadObjectUri(objectName string) string {
	return c.Generate_Download_Object_Uri(c.bucketName, objectName)
}

//func (c *FDSClient) Put(objectName string, data []byte, contentType string, headers *map[string]string) (*Model.PutObjectResult, error) {
//	return c.Put_Object(c.bucketName, objectName, data, contentType, headers)
//}

func init() {
	fdsConf = conf.AppConf.Section("fds")
}

func Newfds(fdsName string) *FDSClient {
	if fdsClient, ok := fdsMap.Load(fdsName); ok {
		return fdsClient.(*FDSClient)
	}

	mu.Lock()
	defer mu.Unlock()
	if fdsClient, ok := fdsMap.Load(fdsName); ok {
		return fdsClient.(*FDSClient)
	}

	AccessKey := fdsConf.Key(fdsName + ".access_key").String()
	AccessSecret := fdsConf.Key(fdsName + ".access_secret").String()
	BucketName := fdsConf.Key(fdsName + ".bucket").String()
	RegionName := fdsConf.Key(fdsName + ".region_name").String()
	EnableHttps := fdsConf.Key(fdsName + ".enable_https").MustBool(true)
	EnableCDN := fdsConf.Key(fdsName + ".enable_cdn").MustBool(true)

	if AccessKey == "" || AccessSecret == "" {
		panic(fmt.Sprintf("FDSConf load failed: %s", fdsName))
	}

	fdsClient := &FDSClient{
		BucketName,
		galaxy_fds_sdk_golang.NEWFDSClient(AccessKey, AccessSecret, RegionName, "", EnableHttps, EnableCDN),
	}

	fdsMap.Store(fdsName, fdsClient)
	return fdsClient
}
