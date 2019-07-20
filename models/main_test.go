package models

import (
	"database/sql"
	"os"
	"testing"

	"github.com/udovin/solve/config"
)

var db *sql.DB

func setupMain() {
	cfg := config.DatabaseConfig{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
	}
	var err error
	db, err = cfg.CreateDB()
	if err != nil {
		os.Exit(1)
	}
}

func teardownMain() {
	_ = db.Close()
}

func setup(t *testing.T) {
	_, err := db.Exec(
		`CREATE TABLE "test_mock_change"` +
			` ("change_id" INTEGER PRIMARY KEY,` +
			` "change_type" INT8,` +
			` "change_time" BIGINT,` +
			` "id" INTEGER,` +
			` "value" VARCHAR(255))`,
	)
	if err != nil {
		t.Error(err)
	}
}

func teardown(t *testing.T) {
	_, err := db.Exec(`DROP TABLE "test_mock_change"`)
	if err != nil {
		t.Error(err)
	}
}

func TestMain(m *testing.M) {
	setupMain()
	defer teardownMain()
	os.Exit(m.Run())
}
