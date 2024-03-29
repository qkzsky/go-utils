package database

import (
	"database/sql/driver"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/qkzsky/go-utils/config"
	"github.com/qkzsky/go-utils/logger"
	"go.uber.org/zap"
	"gopkg.in/ini.v1"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"
)

const DefaultCharset = "utf8"
const DefaultSSLMode = "disable"

var defaultMysqlMaxIdle = runtime.NumCPU() + 1
var defaultMysqlMaxOpen = runtime.NumCPU()*2 + 1

var (
	dbConf *ini.Section
	dbMap  sync.Map
	mu     sync.Mutex
)

func init() {
	dbConf = config.Section("database")
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
	dbName := dbConf.Key(databaseName + ".db").String()
	sslMode := dbConf.Key(databaseName + ".sslmode").MustString(DefaultSSLMode)
	charset := dbConf.Key(databaseName + ".charset").MustString(DefaultCharset)
	maxOpen := dbConf.Key(databaseName + ".max_open").MustInt(defaultMysqlMaxOpen)
	maxIdle := dbConf.Key(databaseName + ".max_idle").MustInt(defaultMysqlMaxIdle)

	var dsn string
	switch drive {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=true&loc=Local&timeout=15s",
			username, password, host, port, dbName, charset)
	case "postgresql":
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

	gormConf := config.Section("gorm")

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
	db.SetLogger(gLogger{logger.NewLogger(config.AppName + "-gorm")})

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
func (l gLogger) Print(values ...interface{}) {
	if len(values) <= 1 {
		return
	}

	var messages []zap.Field
	var (
		sql             string
		msg             string
		formattedValues []string
		level           = values[0].(string)
		source          = fmt.Sprintf("%v", values[1])
	)

	if level == "sql" {
		// duration
		messages = append(messages, zap.String("elapsed", values[2].(time.Duration).String()))

		// sql
		for _, value := range values[4].([]interface{}) {
			indirectValue := reflect.Indirect(reflect.ValueOf(value))
			if indirectValue.IsValid() {
				value = indirectValue.Interface()
				if t, ok := value.(time.Time); ok {
					if t.IsZero() {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", "0000-00-00 00:00:00"))
					} else {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
					}
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
					switch value.(type) {
					case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
						formattedValues = append(formattedValues, fmt.Sprintf("%v", value))
					default:
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
					}
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
				if index < formattedValuesLength && formattedValues != nil {
					sql += formattedValues[index]
				}
			}
		}

		//messages = append(messages, zap.String("sql", sql))
		messages = append(messages, zap.Int64("rows affected or returned", values[5].(int64)))
	} else {
		if strings.HasPrefix(level, "/") {
			source = values[0].(string)
			msg = fmt.Sprintf("%v", values[1:])
		} else {
			msg = fmt.Sprintf("%v", values[2:])
		}
	}

	messages = append(messages, zap.String("source", source))

	switch level {
	case "sql":
		l.Debug(fmt.Sprintf("[gorm] %s: %s", level, sql), messages...)
	default:
		l.Error(fmt.Sprintf("[gorm] %s: %s", level, msg), messages...)
	}
}
