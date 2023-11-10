package migrations

import (
	"github.com/udovin/solve/internal/db"
)

var (
	Schema = db.NewMigrationGroup()
	Data   = db.NewMigrationGroup()
)
