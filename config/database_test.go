package config

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
)

func TestDatabaseConfig_UnmarshalJSON_SQLite(t *testing.T) {
	expectedConfig := DB{
		Driver:  SQLiteDriver,
		Options: SQLiteOptions{Path: "test.sql"},
	}
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
	expectedConfig := DB{
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
	expectedConfig := DB{
		Driver:  "Unsupported",
		Options: nil,
	}
	data, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Fatal(err)
	}
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

func TestDatabaseConfig_UnmarshalJSON_InvalidSQLiteOptions(t *testing.T) {
	expectedConfig := DB{
		Driver:  SQLiteDriver,
		Options: "Invalid",
	}
	data, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Fatal(err)
	}
	var config DB
	if err := json.Unmarshal(data, &config); err == nil {
		t.Fatal("Expected error")
	}
}

func TestDatabaseConfig_UnmarshalJSON_InvalidPostgresOptions(t *testing.T) {
	expectedConfig := DB{
		Driver:  PostgresDriver,
		Options: "Invalid",
	}
	data, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Fatal(err)
	}
	var config DB
	if err := json.Unmarshal(data, &config); err == nil {
		t.Fatal("Expected error")
	}
}

func TestDatabaseConfig_CreateDB_SQLite(t *testing.T) {
	config := DB{
		Driver:  SQLiteDriver,
		Options: SQLiteOptions{Path: "?mode=memory"},
	}
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
	config := DB{
		Driver: PostgresDriver,
		Options: PostgresOptions{
			Password: Secret{Type: DataSecret, Data: ""},
		},
	}
	if _, err := config.Create(); err != nil {
		t.Fatal(err)
	}
	config.Options = PostgresOptions{
		Password: Secret{Type: "Unsupported"},
	}
	if _, err := config.Create(); err == nil {
		t.Fatal("Expected error")
	}
}

func TestDatabaseConfig_CreateDB_Unsupported(t *testing.T) {
	config := DB{
		Driver:  "Unsupported",
		Options: nil,
	}
	if _, err := config.Create(); err == nil {
		t.Fatal("Expected error")
	}
}

func TestCreateSQLiteDB(t *testing.T) {
	if _, err := fixCreateSQLiteDB(nil, sql.ErrConnDone); err == nil {
		t.Fatal("Expected error")
	}
	db, err := sql.Open("sqlite3", "")
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := db.Close(); err != nil {
		t.Error("Error:", err)
	}
	if _, err := fixCreateSQLiteDB(db, nil); err == nil {
		t.Fatal("Expected error")
	}
}
