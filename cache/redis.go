package cache

import (
	"github.com/qkzsky/go-utils/config"
	"gopkg.in/ini.v1"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
)

const (
	DefaultConnectTimeout = 100 * time.Millisecond
	DefaultReadTimeout    = 1000 * time.Millisecond
	DefaultWriteTimeout   = 1000 * time.Millisecond

	DefaultRedisMaxIdle = 10
	DefaultRedisMaxOpen = 20
)

var (
	redisConf *ini.Section
	redisMap  sync.Map
	mu        sync.Mutex
)

func init() {
	redisConf = config.Section("redis")
}

func NewRedis(redisName string) *redis.Pool {
	if pool, ok := redisMap.Load(redisName); ok {
		return pool.(*redis.Pool)
	}

	mu.Lock()
	defer mu.Unlock()
	if pool, ok := redisMap.Load(redisName); ok {
		return pool.(*redis.Pool)
	}

	var err error
	host := redisConf.Key(redisName + ".host").String()
	port := redisConf.Key(redisName + ".port").String()
	auth := redisConf.Key(redisName + ".auth").String()

	if host == "" || port == "" {
		panic("redis config " + redisName + " not found")
	}

	maxOpen := DefaultRedisMaxOpen
	if redisConf.HasKey(redisName + ".max_open") {
		maxOpen, err = redisConf.Key(redisName + ".max_open").Int()
		if err != nil {
			panic(err)
		}
	}

	maxIdle := DefaultRedisMaxIdle
	if redisConf.HasKey(redisName + ".max_idle") {
		maxIdle, err = redisConf.Key(redisName + ".max_idle").Int()
		if err != nil {
			panic(err)
		}
	}

	pool := &redis.Pool{
		MaxIdle:     maxIdle,
		MaxActive:   maxOpen,
		IdleTimeout: 180 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(
				"tcp",
				host+":"+port,
				redis.DialConnectTimeout(DefaultConnectTimeout),
				redis.DialReadTimeout(DefaultReadTimeout),
				redis.DialWriteTimeout(DefaultWriteTimeout),
			)
			if err != nil {
				return nil, err
			}
			if auth != "" {
				if _, err := c.Do("AUTH", auth); err != nil {
					if err := c.Close(); err != nil {
						return nil, err
					}
					return nil, err
				}
			}
			return c, err
		},
		Wait: true,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Second {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}

	redisMap.Store(redisName, pool)
	return pool
}
