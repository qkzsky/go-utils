package logger

import (
	"apollo_cron/utils/conf"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logPath   string
	loggerMap sync.Map
	mu        sync.Mutex

	AppLogger *zap.Logger
)

func init() {
	var err error
	logPath = conf.AppConf.Section("log").Key("path").String()

	if err = os.Mkdir(logPath, os.ModePerm); err != nil {
		if !os.IsExist(err) {
			panic(err)
		}
	}

	AppLogger = NewLogger("app")
}

func GetPath() string {
	return logPath
}

func TimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

func NewLogger(logName string) *zap.Logger {
	if logger, ok := loggerMap.Load(logName); ok {
		return logger.(*zap.Logger)
	}

	mu.Lock()
	defer mu.Unlock()
	if logger, ok := loggerMap.Load(logName); ok {
		return logger.(*zap.Logger)
	}

	var logLevel zap.AtomicLevel
	outputPaths := []string{fmt.Sprintf("%s/%s.log", logPath, logName)}

	var encoderConfig zapcore.EncoderConfig
	var encoding string
	if gin.IsDebugging() {
		logLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
		encoding = "console"
		if flag.Lookup("test.v") == nil {
			outputPaths = append(outputPaths, "stdout")
		}

		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		logLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
		encoding = "json"

		encoderConfig = zap.NewProductionEncoderConfig()
	}
	encoderConfig.EncodeTime = TimeEncoder

	CustomConfig := zap.Config{
		Level:             logLevel,
		Development:       gin.IsDebugging(),
		DisableStacktrace: !gin.IsDebugging(),
		Encoding:          encoding,
		EncoderConfig:     encoderConfig,
		OutputPaths:       outputPaths,
		ErrorOutputPaths:  []string{"stderr"},
	}

	logger, err := CustomConfig.Build()
	if err != nil {
		panic(err)
	}
	if err := logger.Sync(); err != nil {
		//panic(err)
	}

	loggerMap.Store(logName, logger)
	return logger
}
