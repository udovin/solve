package managers

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/models"
)

type FileManager struct {
	Files *models.FileStore
	Dir   string
}

func NewFileManager(core *core.Core) *FileManager {
	return &FileManager{
		Files: core.Files,
		Dir:   core.Config.Storage.FilesDir,
	}
}

func (m *FileManager) UploadFile(ctx context.Context, formFile *multipart.FileHeader) (Future[models.File], error) {
	if tx := db.GetTx(ctx); tx != nil {
		return nil, fmt.Errorf("cannot upload file in transaction")
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(10 * time.Minute)
	}
	filePath, err := m.generatePath(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot generate path: %w", err)
	}
	file := models.File{
		Status:     models.PendingFile,
		ExpireTime: models.NInt64(deadline.Unix()),
		Name:       formFile.Filename,
		Size:       formFile.Size,
		Path:       filePath,
	}
	src, err := formFile.Open()
	if err != nil {
		return nil, err
	}
	if err := m.Files.Create(ctx, &file); err != nil {
		_ = src.Close()
		return nil, err
	}
	return Async(func() (models.File, error) {
		defer src.Close()
		systemDir := filepath.Join(m.Dir, filepath.FromSlash(path.Dir(filePath)))
		if err := os.MkdirAll(systemDir, 0777); err != nil {
			return models.File{}, err
		}
		dst, err := os.Create(filepath.Join(m.Dir, filepath.FromSlash(filePath)))
		if err != nil {
			return models.File{}, err
		}
		defer dst.Close()
		written, err := io.Copy(dst, src)
		if err != nil {
			return models.File{}, err
		}
		file.Size = written
		return file, nil
	}), nil
}

func (m *FileManager) ConfirmFile(ctx context.Context, file *models.File) error {
	clone := *file
	clone.Status = models.AvailableFile
	clone.ExpireTime = 0
	if err := m.Files.Update(ctx, clone); err != nil {
		return err
	}
	*file = clone
	return nil
}

func (m *FileManager) generatePath(ctx context.Context) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	filePath := path.Join(
		hex.EncodeToString(bytes[0:3]),
		hex.EncodeToString(bytes[3:6]),
		hex.EncodeToString(bytes[6:9]),
		hex.EncodeToString(bytes[9:]),
	)
	systemPath := filepath.Join(m.Dir, filepath.FromSlash(filePath))
	if _, err := os.Stat(systemPath); err == nil {
		return "", os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return filePath, nil
}
