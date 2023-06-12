package managers

import (
	"bytes"
	"context"
	"testing"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/migrations"
	"github.com/udovin/solve/models"
)

func TestFileManager(t *testing.T) {
	c, err := core.NewCore(config.Config{
		DB: config.DB{
			Options: config.SQLiteOptions{Path: ":memory:"},
		},
		Storage: &config.Storage{
			Options: config.LocalStorageOptions{
				FilesDir: t.TempDir(),
			},
		},
	})
	if err != nil {
		t.Fatal("Error:", err)
	}
	c.SetupAllStores()
	if err := db.ApplyMigrations(context.Background(), c.DB, "solve", migrations.Schema); err != nil {
		t.Fatal("Error:", err)
	}
	c.Start()
	defer c.Stop()
	manager := NewFileManager(c)
	file, err := manager.UploadFile(
		context.Background(),
		&FileReader{
			Reader: bytes.NewBufferString("test"),
		},
	)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := manager.ConfirmUploadFile(context.Background(), &file); err != nil {
		t.Fatal("Error:", err)
	}
	if err := manager.DeleteFile(models.WithSync(context.Background()), file.ID); err != nil {
		t.Fatal("Error:", err)
	}
}
