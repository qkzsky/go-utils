package database

import (
	"go-utils/conf"
	"go-utils/logger"
	"database/sql/driver"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"go.uber.org/zap"
	"gopkg.in/ini.v1"
	"micode.be.xiaomi.com/systech/asset/xmcrypt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const DefaultCharset = "utf8"
const DefaultSSLMode = "disable"
const DefaultMysqlMaxIdle = 10
const DefaultMysqlMaxOpen = 20

const CryptKey = "jBhXz9AsRcpKTnMdVgHr2tE5G86JuOlb"

var (
	dbConf *ini.Section
	dbMap  sync.Map
	mu     sync.Mutex
)

func init() {
	dbConf = conf.AppConf.Section("database")
}

func NewDB(databaseName string) *gorm.DB {
	if db, ok := dbMap.Load(databaseName); ok {
		return db.(*gorm.DB)
	}

	mu.Lock()
	defer mu.Unlock()
	if db, ok := dbMap.Load(databaseName); ok {
		return db.(*gorm.DB)
	}

	var err error
	drive := dbConf.Key(databaseName + ".drive").String()
	host := dbConf.Key(databaseName + ".host").String()
	port := dbConf.Key(databaseName + ".port").String()
	username := dbConf.Key(databaseName + ".username").String()
	password := dbConf.Key(databaseName + ".password").String()
	if passwordDec, err := xmcrypt.DecryptExtend(password, CryptKey); err == nil {
		password = passwordDec
	}
	dbName := dbConf.Key(databaseName + ".db").String()
	sslMode := dbConf.Key(databaseName + ".disable").MustString(DefaultSSLMode)
	charset := dbConf.Key(databaseName + ".charset").MustString(DefaultCharset)
	maxOpen := dbConf.Key(databaseName + ".max_open").MustInt(DefaultMysqlMaxOpen)
	maxIdle := dbConf.Key(databaseName + ".max_idle").MustInt(DefaultMysqlMaxIdle)

	var dsn string
	switch drive {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=false&loc=Local&timeout=15s",
			username, password, host, port, dbName, charset)
	case "postgres":
		dsn = fmt.Sprintf("host=%s:%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, username, password, dbName, sslMode)
	default:
		panic(fmt.Errorf("unknown database drive: %s", drive))
	}

	db, err := gorm.Open(drive, dsn)
	if err != nil {
		panic(err)
	}

	err = db.DB().Ping()
	if err != nil {
		panic(err)
	}

	// 连接及连接池配置
	db.DB().SetConnMaxLifetime(2 * time.Hour)
	db.DB().SetMaxOpenConns(maxOpen)
	db.DB().SetMaxIdleConns(maxIdle)

	gormConf := conf.AppConf.Section("gorm")

	logModeCfg := gormConf.Key("log.mode")
	if logModeCfg.String() != "" {
		logMode, err := logModeCfg.Bool()
		if err != nil {
			panic(err)
		}
		db.LogMode(logMode)
	}

	//var logFile *os.File
	//fileName := logger.GetPath() + "/" + gormConf.Key("log.file").String()
	//logFile, err = os.OpenFile(fileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	//if err != nil {
	//	panic(err)
	//}
	//db.SetLogger(gLogger{log.New(logFile, "", 0)})
	db.SetLogger(gLogger{logger.NewLogger("gorm")})

	dbMap.Store(databaseName, db)

	return db
}

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

type gLogger struct {
	*zap.Logger
}

// Print format & print log
func (logger gLogger) Print(values ...interface{}) {
	if len(values) <= 1 {
		return
	}

	var messages []zap.Field
	var (
		sql             string
		formattedValues []string
		level           = values[0].(string)
		source          = fmt.Sprintf("%v", values[1])
	)

	if level == "sql" {
		// duration
		messages = append(messages, zap.Duration("elapsed", values[2].(time.Duration)))

		// sql
		for _, value := range values[4].([]interface{}) {
			indirectValue := reflect.Indirect(reflect.ValueOf(value))
			if indirectValue.IsValid() {
				value = indirectValue.Interface()
				if t, ok := value.(time.Time); ok {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
				} else if b, ok := value.([]byte); ok {
					if str := string(b); isPrintable(str) {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", str))
					} else {
						formattedValues = append(formattedValues, "'<binary>'")
					}
				} else if r, ok := value.(driver.Valuer); ok {
					if value, err := r.Value(); err == nil && value != nil {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
					} else {
						formattedValues = append(formattedValues, "NULL")
					}
				} else {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
				}
			} else {
				formattedValues = append(formattedValues, "NULL")
			}
		}

		// differentiate between $n placeholders or else treat like ?
		if regexp.MustCompile(`\$\d+`).MatchString(values[3].(string)) {
			sql = values[3].(string)
			for index, value := range formattedValues {
				placeholder := fmt.Sprintf(`\$%d([^\d]|$)`, index+1)
				sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value+"$1")
			}
		} else {
			formattedValuesLength := len(formattedValues)
			for index, value := range regexp.MustCompile(`\?`).Split(values[3].(string), -1) {
				sql += value
				if index < formattedValuesLength {
					sql += formattedValues[index]
				}
			}
		}

		messages = append(messages, zap.String("sql", sql))
		messages = append(messages, zap.String("rows affected or returned", strconv.FormatInt(values[5].(int64), 10)))
	} else {
		if strings.HasPrefix(level, "/") {
			source = values[0].(string)
			messages = append(messages, zap.String("message", fmt.Sprintf("%v", values[1:])))
		} else {
			messages = append(messages, zap.String("message", fmt.Sprintf("%v", values[2:])))
		}
	}

	messages = append(messages, zap.String("source", source))

	switch level {
	case "sql":
		logger.Debug("[gorm] sql", messages...)
	case "log":
		logger.Info("[gorm] log", messages...)
	case "error":
		logger.Error("[gorm] error", messages...)
	default:
		logger.Error("[gorm] error", messages...)
	}
}
