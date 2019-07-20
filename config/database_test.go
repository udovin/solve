package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDatabaseConfig_UnmarshalJSON_SQLite(t *testing.T) {
	expectedConfig := DatabaseConfig{
		Driver:  SQLiteDriver,
		Options: SQLiteOptions{Path: "test.sql"},
	}
	data, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Error(err)
	}
	var config DatabaseConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(expectedConfig, config) {
		t.Error("Configs are not equal")
	}
}

func TestDatabaseConfig_UnmarshalJSON_Postgres(t *testing.T) {
	expectedConfig := DatabaseConfig{
		Driver: PostgresDriver,
		Options: PostgresOptions{
			Host:     "localhost",
			User:     "user",
			Password: Secret{Type: DataSecret, Data: "password"},
			Name:     "database",
		},
	}
	data, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Error(err)
	}
	var config DatabaseConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(expectedConfig, config) {
		t.Error("Configs are not equal")
	}
}

func TestDatabaseConfig_UnmarshalJSON_Unsupported(t *testing.T) {
	expectedConfig := DatabaseConfig{
		Driver:  "Unsupported",
		Options: nil,
	}
	data, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Error(err)
	}
	var config DatabaseConfig
	if err := json.Unmarshal(data, &config); err == nil {
		t.Error("Expected error")
	}
}

func TestDatabaseConfig_CreateDB_SQLite(t *testing.T) {
	config := DatabaseConfig{
		Driver:  SQLiteDriver,
		Options: SQLiteOptions{Path: "?mode=memory"},
	}
	db, err := config.CreateDB()
	if err != nil {
		t.Error(err)
	}
	if err := db.Ping(); err != nil {
		t.Error(err)
	}
	_ = db.Close()
}
