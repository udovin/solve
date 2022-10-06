package migrations

import (
	"github.com/udovin/solve/db"
)

var (
	Schema = db.NewMigrationGroup()
	Data   = db.NewMigrationGroup()
)
