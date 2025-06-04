package db

import (
	"github.com/go-playground/validator/v10"
	"testing"
	"time"
)

func TestDatabaseConfigValidation(t *testing.T) {
	validConfig := &ConnectionConfig{
		DbType:       Mysql,
		Host:         "localhost",
		Port:         3306,
		Username:     "root",
		Password:     "",
		DbName:       "mydb",
		Timeout:      3 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	invalidConfig := &ConnectionConfig{
		DbType:       "invalid_db_type",
		Host:         "",
		Port:         -1,
		Username:     "",
		Password:     "",
		DbName:       "",
		MaxIdleConns: 0,
		MaxOpenConns: 0,
	}

	validate := validator.New()

	if err := validate.Struct(validConfig); err != nil {
		t.Errorf("validation of valid database config failed: %v", err)
	}

	if err := validate.Struct(invalidConfig); err == nil {
		t.Error("validation of invalid database config passed unexpectedly")
	}
}

func TestCustomValidationTags(t *testing.T) {
	type CustomStruct struct {
		CustomField string `validate:"custom"`
	}

	validate := validator.New()
	_ = validate.RegisterValidation("custom", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return value == "foo" || value == "bar"
	})

	validStruct := &CustomStruct{CustomField: "foo"}
	invalidStruct := &CustomStruct{CustomField: "baz"}

	if err := validate.Struct(validStruct); err != nil {
		t.Errorf("validation of valid custom struct failed: %v", err)
	}

	if err := validate.Struct(invalidStruct); err == nil {
		t.Error("validation of invalid custom struct passed unexpectedly")
	}
}

func TestConnectionConfig_Dsn(t *testing.T) {
	tests := []struct {
		name    string
		config  ConnectionConfig
		wantDsn string
	}{
		{
			name: "MySQL connection",
			config: ConnectionConfig{
				DbType:   Mysql,
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: "password",
				DbName:   "test",
				Timeout:  3 * time.Second,
			},
			wantDsn: "root:password@tcp(localhost:3306)/test?parseTime=true&timeout=3s",
		},
		{
			name: "Postgres connection",
			config: ConnectionConfig{
				DbType:   Postgresql,
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "password",
				DbName:   "test",
				Timeout:  3 * time.Second,
			},
			wantDsn: "user=postgres password=password host=localhost port=5432 dbname=test sslmode=disable connect_timeout=3",
		},
		{
			name: "Different port",
			config: ConnectionConfig{
				DbType:   Mysql,
				Host:     "localhost",
				Port:     8080,
				Username: "root",
				Password: "password",
				DbName:   "test",
				Timeout:  5 * time.Second,
			},
			wantDsn: "root:password@tcp(localhost:8080)/test?parseTime=true&timeout=5s",
		},
		{
			name: "All configurations are empty",
			config: ConnectionConfig{
				DbType:   "",
				Host:     "",
				Port:     0,
				Username: "",
				Password: "",
				DbName:   "",
				Timeout:  0 * time.Second,
			},
			wantDsn: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDsn := tt.config.Dsn()
			if gotDsn != tt.wantDsn {
				t.Errorf("ConnectionConfig.Dsn() = %v, want %v", gotDsn, tt.wantDsn)
			}
		})
	}
}
