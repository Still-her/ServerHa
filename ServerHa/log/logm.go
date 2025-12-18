package log

import (
	"ha/config"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var logm *zap.SugaredLogger

func initm() {
	logm = makelogm()
	logm.Info("logm init success")
	defer logm.Sync()
}

func makelogm() *zap.SugaredLogger {
	atom := zap.NewAtomicLevelAt(zapcore.Level(config.Instance.Logs.Loglevel))
	writeSyncer := getLogWriter(config.Instance.Logs.Logmfile)
	encoder := getEncoder(true)
	core := zapcore.NewCore(encoder, writeSyncer, atom)
	logger := zap.New(core, zap.AddCaller())
	logm = logger.Sugar()
	return logm
}

func getEncoder(islevel bool) zapcore.Encoder {
	customTimeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	customLevelEncoder := func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		if true == islevel {
			enc.AppendString(level.CapitalString())
		} else {
			enc.AppendString("")
		}
	}
	customCallerEncoder := func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString("[" + caller.TrimmedPath() + "]")
		//enc.AppendString("")
	}
	zapLoggerEncoderConfig := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "caller",
		MessageKey:       "message",
		StacktraceKey:    "stacktrace",
		EncodeCaller:     customCallerEncoder,
		EncodeTime:       customTimeEncoder,
		EncodeLevel:      customLevelEncoder,
		EncodeDuration:   zapcore.SecondsDurationEncoder,
		LineEnding:       "\n",
		ConsoleSeparator: " ",
	}
	return zapcore.NewConsoleEncoder(zapLoggerEncoderConfig)
}

func getLogWriter(logf string) zapcore.WriteSyncer {

	lumberJackLogger := &lumberjack.Logger{
		Filename:   logf,                         // ⽇志⽂件路径
		MaxSize:    config.Instance.Logs.Maxsize, // 1M=1024KB=1024000byte
		MaxBackups: config.Instance.Logs.Maxback, // 保留旧文件的最大个数
		MaxAge:     config.Instance.Logs.Maxdays, // 保留旧文件的最大天数
		LocalTime:  true,                         //是否使用本地时间，否则使用UTC时间
		Compress:   false,                        // 是否压缩 disabled by default
	}
	return zapcore.AddSync(lumberJackLogger)
}

func Debug(args ...interface{}) {
	logm.Debug(args)
}

func Info(args ...interface{}) {
	logm.Info(args)
}

func Warn(args ...interface{}) {
	logm.Warn(args)
}

func Error(args ...interface{}) {
	logm.Error(args)
}
