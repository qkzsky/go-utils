package redis

import (
	"github.com/qkzsky/go-utils/config"
	"gopkg.in/ini.v1"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
)

const (
	DefaultConnectTimeout = 100 * time.Millisecond
	DefaultReadTimeout    = 1000 * time.Millisecond
	DefaultWriteTimeout   = 1000 * time.Millisecond
)

var (
	defaultIdleSize = runtime.NumCPU() + 1
	defaultPoolSize = runtime.NumCPU()*2 + 1
)

var (
	redisConf *ini.Section
	redisMap  sync.Map
	mu        sync.Mutex
)

func init() {
	redisConf = config.Section("redis")
}

func NewRedis(redisName string) *redis.Client {
	if client, ok := redisMap.Load(redisName); ok {
		return client.(*redis.Client)
	}

	mu.Lock()
	defer mu.Unlock()
	if client, ok := redisMap.Load(redisName); ok {
		return client.(*redis.Client)
	}

	var err error
	host := redisConf.Key(redisName + ".host").String()
	port := redisConf.Key(redisName + ".port").String()
	auth := redisConf.Key(redisName + ".auth").String()

	if host == "" || port == "" {
		panic("redis config " + redisName + " not found")
	}

	poolSize := defaultPoolSize
	if redisConf.HasKey(redisName + ".max_open") {
		poolSize, err = redisConf.Key(redisName + ".max_open").Int()
		if err != nil {
			panic(err)
		}
	}

	idleSize := defaultIdleSize
	if redisConf.HasKey(redisName + ".max_idle") {
		idleSize, err = redisConf.Key(redisName + ".max_idle").Int()
		if err != nil {
			panic(err)
		}
	}

	client := redis.NewClient(&redis.Options{
		Network:      "tcp",
		Addr:         host + ":" + port,
		Password:     auth,
		DialTimeout:  DefaultConnectTimeout,
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		PoolSize:     poolSize,
		MinIdleConns: idleSize,
		IdleTimeout:  180 * time.Second,
	})
	if client.Ping().Err() != nil {
		log.Fatalln("[redis] " + err.Error())
	}

	redisMap.Store(redisName, client)
	return client
}
