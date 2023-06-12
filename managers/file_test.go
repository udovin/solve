package managers

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
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
		&FileReader{Reader: bytes.NewReader([]byte("test"))},
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

func TestFileManagerS3(t *testing.T) {
	fakeS3Mem := s3mem.New()
	fakeS3Mem.CreateBucket("test-bucket")
	fakeS3 := gofakes3.New(fakeS3Mem)
	fakeS3Server := httptest.NewServer(fakeS3.Server())
	defer fakeS3Server.Close()
	c, err := core.NewCore(config.Config{
		DB: config.DB{
			Options: config.SQLiteOptions{Path: ":memory:"},
		},
		Storage: &config.Storage{
			Options: config.S3StorageOptions{
				Endpoint:        fakeS3Server.URL,
				Bucket:          "test-bucket",
				PathPrefix:      "files/",
				AccessKeyID:     "test",
				SecretAccessKey: "test",
				UsePathStyle:    true,
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
			Reader: bytes.NewReader([]byte("test")),
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
