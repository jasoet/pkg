package db

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseType string

const (
	Mysql      DatabaseType = "MYSQL"
	Postgresql DatabaseType = "POSTGRES"
	MSSQL      DatabaseType = "MSSQL"
)

type ConnectionConfig struct {
	DbType       DatabaseType  `yaml:"dbType" validate:"required,oneof=MYSQL POSTGRES MSSQL" mapstructure:"dbType"`
	Host         string        `yaml:"host" validate:"required,min=1" mapstructure:"host"`
	Port         int           `yaml:"port" mapstructure:"port"`
	Username     string        `yaml:"username" validate:"required,min=1" mapstructure:"username"`
	Password     string        `yaml:"password" mapstructure:"password"`
	DbName       string        `yaml:"dbName" validate:"required,min=1" mapstructure:"dbName"`
	Timeout      time.Duration `yaml:"timeout" mapstructure:"timeout" validate:"min=3s"`
	MaxIdleConns int           `yaml:"maxIdleConns" mapstructure:"maxIdleConns" validate:"min=1"`
	MaxOpenConns int           `yaml:"maxOpenConns" mapstructure:"maxOpenConns" validate:"min=2"`
}

func (c *ConnectionConfig) Dsn() string {
	timeoutString := fmt.Sprintf("%ds", c.Timeout/time.Second)

	var dsn string
	switch c.DbType {
	case Mysql:
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=%s", c.Username, c.Password, c.Host, c.Port, c.DbName, timeoutString)
	case Postgresql:
		dsn = fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=disable connect_timeout=%d", c.Username, c.Password, c.Host, c.Port, c.DbName, int(c.Timeout.Seconds()))
	case MSSQL:
		dsn = fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s&connectTimeout=%s&encrypt=disable", c.Username, c.Password, c.Host, c.Port, c.DbName, timeoutString)
	}

	return dsn
}

func (c *ConnectionConfig) Pool() (*gorm.DB, error) {
	if c.Dsn() == "" {
		return nil, fmt.Errorf("dsn is empty")
	}

	var dialector gorm.Dialector
	switch c.DbType {
	case Mysql:
		dialector = mysql.Open(c.Dsn())
	case Postgresql:
		dialector = postgres.Open(c.Dsn())
	case MSSQL:
		dialector = sqlserver.Open(c.Dsn())
	default:
		return nil, fmt.Errorf("unsupported database type: %s", c.DbType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.MaxOpenConns)

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func (c *ConnectionConfig) SqlDB() (*sql.DB, error) {
	gormDB, err := c.Pool()
	if err != nil {
		return nil, err
	}

	return gormDB.DB()
}
