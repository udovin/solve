package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDatabaseConfig_UnmarshalJSON_SQLite(t *testing.T) {
	expectedConfig := DB{Options: SQLiteOptions{Path: "test.sql"}}
	data, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Fatal(err)
	}
	var config DB
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expectedConfig, config) {
		t.Fatal("Configs are not equal")
	}
}

func TestDatabaseConfig_UnmarshalJSON_Postgres(t *testing.T) {
	expectedConfig := DB{Options: PostgresOptions{
		Hosts:    []string{"localhost:6432"},
		User:     "user",
		Password: "password",
		Name:     "database",
	}}
	data, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Fatal(err)
	}
	var config DB
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expectedConfig, config) {
		t.Fatal("Configs are not equal")
	}
}

func TestDatabaseConfig_UnmarshalJSON_Unsupported(t *testing.T) {
	data := []byte(`{"driver": "unsupported", "options": {}}`)
	var config DB
	if err := json.Unmarshal(data, &config); err == nil {
		t.Fatal("Expected error")
	}
}

func TestDatabaseConfig_UnmarshalJSON_Invalid(t *testing.T) {
	var config DB
	if err := json.Unmarshal([]byte("[]"), &config); err == nil {
		t.Fatal("Expected error")
	}
}

func TestDatabaseConfig_MarshalJSON_Invalid(t *testing.T) {
	expectedConfig := DB{Options: "Invalid"}
	if _, err := json.Marshal(expectedConfig); err == nil {
		t.Fatal("Expected error")
	}
}

func TestDatabaseConfig_UnmarshalJSON_InvalidSQLiteOptions(t *testing.T) {
	data := []byte(`{"driver": "sqlite", "options": "invalid"}`)
	var config DB
	if err := json.Unmarshal(data, &config); err == nil {
		t.Fatal("Expected error")
	}
}

func TestDatabaseConfig_UnmarshalJSON_InvalidPostgresOptions(t *testing.T) {
	data := []byte(`{"driver": "postgres", "options": "invalid"}`)
	var config DB
	if err := json.Unmarshal(data, &config); err == nil {
		t.Fatal("Expected error")
	}
}

func TestDatabaseConfig_CreateDB_SQLite(t *testing.T) {
	config := DB{Options: SQLiteOptions{Path: ":memory:"}}
	db, err := config.Create()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
}

func TestDatabaseConfig_CreateDB_Postgres(t *testing.T) {
	config := DB{Options: PostgresOptions{
		Password: "",
	}}
	if _, err := config.Create(); err != nil {
		t.Fatal(err)
	}
}

func TestDatabaseConfig_CreateDB_Empty(t *testing.T) {
	config := DB{
		Options: nil,
	}
	if _, err := config.Create(); err == nil {
		t.Fatal("Expected error")
	}
}
