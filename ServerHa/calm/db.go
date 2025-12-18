package calm

import (
	"database/sql"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type GormModel struct {
	Id int64 `gorm:"primaryKey;autoIncrement" json:"id" form:"id"`
}

var (
	db      *gorm.DB
	dbs     *gorm.DB
	dbpg    *gorm.DB
	sqlDB   *sql.DB
	sqlSDB  *sql.DB
	sqlPgDB *sql.DB
)

func OpenDB(dsn string, config *gorm.Config, maxIdleConns, maxOpenConns int, models ...interface{}) (err error) {
	if config == nil {
		config = &gorm.Config{}
	}

	if config.NamingStrategy == nil {
		config.NamingStrategy = schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		}
	}

	if db, err = gorm.Open(mysql.Open(dsn), config); err != nil {
		log.Errorf("opens database failed: %s", err.Error())
		return
	}

	if sqlDB, err = db.DB(); err == nil {
		sqlDB.SetMaxIdleConns(maxIdleConns)
		sqlDB.SetMaxOpenConns(maxOpenConns)
	} else {
		log.Error(err)
	}

	if err = db.AutoMigrate(models...); nil != err {
		log.Errorf("auto migrate tables failed: %s", err.Error())
	}
	return
}

func OpenSDB(dsn string, config *gorm.Config, maxIdleConns, maxOpenConns int, models ...interface{}) (err error) {
	if config == nil {
		config = &gorm.Config{}
	}
	if config.NamingStrategy == nil {

		log.Errorf("config.NamingStrategy == nil: %s", config.NamingStrategy)
		config.NamingStrategy = schema.NamingStrategy{
			TablePrefix:   "a_",
			SingularTable: false,
		}
	}
	if dbs, err = gorm.Open(mysql.Open(dsn), config); err != nil {
		log.Errorf("opens database failed: %s", err.Error())
		return
	}
	if sqlSDB, err = dbs.DB(); err == nil {
		sqlSDB.SetMaxIdleConns(maxIdleConns)
		sqlSDB.SetMaxOpenConns(maxOpenConns)
	} else {
		log.Error(err)
	}

	if err = dbs.AutoMigrate(models...); nil != err {
		log.Errorf("auto migrate tables failed: %s", err.Error())
	}
	return
}

func OpenMDB(dsn string, config *gorm.Config, maxIdleConns, maxOpenConns int, models ...interface{}) (mdb *gorm.DB, err error) {
	if config == nil {
		config = &gorm.Config{}
	}

	config.NamingStrategy = schema.NamingStrategy{
		TablePrefix:   "",
		SingularTable: true,
	}

	if config.NamingStrategy == nil {
		config.NamingStrategy = schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		}
	}

	if mdb, err = gorm.Open(postgres.Open(dsn), config); err != nil {
		log.Errorf("opens database failed: %s", err.Error())
		return
	}

	if sqlMDB, err := mdb.DB(); err == nil {
		sqlMDB.SetMaxIdleConns(maxIdleConns)
		sqlMDB.SetMaxOpenConns(maxOpenConns)
	} else {
		log.Error(err)
	}

	if err = mdb.AutoMigrate(models...); nil != err {
		log.Errorf("auto migrate tables failed: %s", err.Error())
	}
	return
}

func OpenPgDB(dsn string, config *gorm.Config, maxIdleConns, maxOpenConns int, models ...interface{}) (err error) {
	if config == nil {
		config = &gorm.Config{}
	}

	config.NamingStrategy = schema.NamingStrategy{
		TablePrefix:   "",
		SingularTable: true,
	}

	if config.NamingStrategy == nil {
		config.NamingStrategy = schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		}
	}

	if dbpg, err = gorm.Open(postgres.Open(dsn), config); err != nil {
		log.Errorf("opens database failed: %s", err.Error())
		return
	}

	if sqlPgDB, err = dbpg.DB(); err == nil {
		sqlPgDB.SetMaxIdleConns(maxIdleConns)
		sqlPgDB.SetMaxOpenConns(maxOpenConns)
	} else {
		log.Error(err)
	}

	if err = dbpg.AutoMigrate(models...); nil != err {
		log.Errorf("auto migrate tables failed: %s", err.Error())
	}
	return
}

// 获取数据库链接
func DB() *gorm.DB {
	return db
}

func SDB() *gorm.DB {
	return dbs
}

func PGDB() *gorm.DB {
	return dbpg
}

// 关闭连接
func CloseDB() {
	if sqlDB == nil {
		return
	}
	if err := sqlDB.Close(); nil != err {
		log.Errorf("Disconnect from database failed: %s", err.Error())
	}
}

// 关闭连接
func CloseSDB() {
	if sqlSDB == nil {
		return
	}
	if err := sqlSDB.Close(); nil != err {
		log.Errorf("Disconnect from database failed: %s", err.Error())
	}
}

// 关闭连接
func ClosePGDB() {
	if sqlPgDB == nil {
		return
	}
	if err := sqlPgDB.Close(); nil != err {
		log.Errorf("Disconnect from database failed: %s", err.Error())
	}
}
