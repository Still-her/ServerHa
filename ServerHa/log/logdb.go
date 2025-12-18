package log

import (
	"context"
	"ha/config"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm/logger"
)

var logd *zap.SugaredLogger

func initd() {
	logd = makelogd()
	defer logd.Sync()
}

func makelogd() *zap.SugaredLogger {
	atom := zap.NewAtomicLevelAt(zapcore.Level(-1))
	writeSyncer := getLogWriter(config.Instance.Logs.Logdfile)
	encoder := getEncoder(false)
	core := zapcore.NewCore(encoder, writeSyncer, atom)
	logger := zap.New(core, zap.AddCaller())
	logd = logger.Sugar()
	return logd
}

type Logdb struct {
	logger.Config
	Dbnode int
}

func (logdb *Logdb) LogMode(level logger.LogLevel) logger.Interface {
	logdb.LogLevel = level
	return logdb
}
func (logdb Logdb) Info(ctx context.Context, msg string, args ...interface{}) {

}

func (logdb Logdb) Warn(ctx context.Context, msg string, args ...interface{}) {

}

func (logdb Logdb) Error(ctx context.Context, msg string, args ...interface{}) {

}

func (logdb Logdb) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	if config.Instance.Logs.Isshowdb {
		if config.Instance.Logs.Isonlyfa {
			if nil != err {
				logd.Error("[bdnode:", logdb.Dbnode, "]", "[time:", elapsed, "]", "[rows:", rows, "]", "[sql:", sql, "]", "[err:", err.Error(), "]")
			}
		} else {
			if nil != err {
				logd.Error("[bdnode:", logdb.Dbnode, "]", "[time:", elapsed, "]", "[rows:", rows, "]", "[sql:", sql, "]", "[err:", err.Error(), "]")
			} else {
				logd.Info("[bdnode:", logdb.Dbnode, "]", "[time:", elapsed, "]", "[rows:", rows, "]", "[sql:", sql, "]")
			}
		}
	}
}

func Newlogdb(dbnode int) logger.Interface {
	log := &Logdb{
		Config: logger.Config{
			SlowThreshold: time.Second,
			Colorful:      true,
			LogLevel:      4,
		},
		Dbnode: dbnode,
	}
	return log
}
