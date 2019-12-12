package logger

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qkzsky/go-utils/config"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logPath   string
	loggerMap sync.Map
	mu        sync.Mutex

	defaultLogger *zap.Logger
)

var levelMap = map[string]zapcore.Level{
	"debug":  zapcore.DebugLevel,
	"info":   zapcore.InfoLevel,
	"warn":   zapcore.WarnLevel,
	"error":  zapcore.ErrorLevel,
	"dpanic": zapcore.DPanicLevel,
	"panic":  zapcore.PanicLevel,
	"fatal":  zapcore.FatalLevel,
}

func getLoggerLevel(lvl string) zapcore.Level {
	if level, ok := levelMap[lvl]; ok {
		return level
	}
	return zapcore.InfoLevel
}

func init() {
	var err error
	logPath = config.Section("log").Key("path").String()

	if err = os.Mkdir(logPath, os.ModePerm); err != nil {
		if !os.IsExist(err) {
			panic(err)
		}
	}

	defaultLogger = NewLogger(config.AppName)
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

	fileName := fmt.Sprintf("%s/%s.log", logPath, logName)
	var logLevel zap.AtomicLevel
	fileWriters := []zapcore.WriteSyncer{zapcore.AddSync(&lumberjack.Logger{
		Filename:  fileName,
		MaxSize:   1 << 10, // MB
		LocalTime: true,
		Compress:  true,
	})}

	var core zapcore.Core
	encoder := zap.NewProductionEncoderConfig()
	//encoder.EncodeTime = TimeEncoder
	encoder.EncodeTime = zapcore.ISO8601TimeEncoder

	if gin.IsDebugging() {
		logLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
		//if flag.Lookup("test.v") == nil {
		//	outputPaths = append(outputPaths, "stdout")
		//}

		// debug 日志输出至日志文件、标准输出
		core = zapcore.NewTee(
			zapcore.NewCore(zapcore.NewJSONEncoder(encoder), zap.CombineWriteSyncers(fileWriters...), logLevel),
			func() zapcore.Core {
				consoleWriter, closeOut, err := zap.Open("stdout")
				if err != nil {
					closeOut()
					panic(err)
				}
				encoder.EncodeLevel = zapcore.CapitalColorLevelEncoder
				return zapcore.NewCore(zapcore.NewConsoleEncoder(encoder), zap.CombineWriteSyncers(consoleWriter), logLevel)
			}(),
		)
	} else {
		logLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
		core = zapcore.NewCore(zapcore.NewJSONEncoder(encoder), zap.CombineWriteSyncers(fileWriters...), logLevel)
	}

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	loggerMap.Store(logName, logger)
	return logger
}

func Debug(msg string, fields ...zap.Field) {
	defaultLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	defaultLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	defaultLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	defaultLogger.Error(msg, fields...)
}

func DPanic(msg string, fields ...zap.Field) {
	defaultLogger.DPanic(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	defaultLogger.Panic(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	defaultLogger.Fatal(msg, fields...)
}
