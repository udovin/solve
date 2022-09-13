package managers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	Files         *models.FileStore
	Dir           string
	UploadTimeout time.Duration
}

func NewFileManager(core *core.Core) *FileManager {
	return &FileManager{
		Files:         core.Files,
		Dir:           core.Config.Storage.FilesDir,
		UploadTimeout: 10 * time.Minute,
	}
}

type FileReader struct {
	Name   string
	Size   int64
	Reader io.Reader
}

func (f *FileReader) Close() error {
	if closer, ok := f.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func NewMultipartFileReader(file *multipart.FileHeader) (*FileReader, error) {
	f := FileReader{
		Name: file.Filename,
		Size: file.Size,
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	f.Reader = reader
	return &f, nil
}

// UploadFile adds file to file storage and starts upload.
//
// You shold call ConfirmUploadFile for marking file available.
func (m *FileManager) UploadFile(ctx context.Context, fileReader *FileReader) (models.File, error) {
	defer func() { _ = fileReader.Close() }()
	if tx := db.GetTx(ctx); tx != nil {
		return models.File{}, fmt.Errorf("cannot upload file in transaction")
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(m.UploadTimeout)
	}
	filePath, err := m.generatePath(ctx)
	if err != nil {
		return models.File{}, fmt.Errorf("cannot generate path: %w", err)
	}
	file := models.File{
		Status:     models.PendingFile,
		ExpireTime: models.NInt64(deadline.Add(time.Minute).Unix()),
		Name:       fileReader.Name,
		Size:       fileReader.Size,
		Path:       filePath,
	}
	if err := m.Files.Create(ctx, &file); err != nil {
		return models.File{}, err
	}
	systemDir := filepath.Join(m.Dir, filepath.FromSlash(path.Dir(filePath)))
	if err := os.MkdirAll(systemDir, 0777); err != nil {
		return models.File{}, err
	}
	dst, err := os.Create(filepath.Join(m.Dir, filepath.FromSlash(filePath)))
	if err != nil {
		return models.File{}, err
	}
	defer func() { _ = dst.Close() }()
	written, err := io.Copy(dst, fileReader.Reader)
	if err != nil {
		return models.File{}, err
	}
	file.Size = written
	return file, nil
}

func (m *FileManager) ConfirmUploadFile(ctx context.Context, file *models.File) error {
	if file.Status != models.PendingFile {
		return fmt.Errorf("file shoud be in pending status")
	}
	clone := *file
	clone.Status = models.AvailableFile
	clone.ExpireTime = 0
	if err := m.Files.Update(ctx, clone); err != nil {
		return err
	}
	*file = clone
	return nil
}

func (m *FileManager) waitFileAvailable(ctx context.Context, file *models.File) error {
	if file.Status == models.AvailableFile {
		return nil
	}
	if file.Status != models.PendingFile {
		return fmt.Errorf("file has invalid status: %s", file.Status)
	}
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	expireTime := time.Unix(int64(file.ExpireTime), 0)
	for file.Status == models.PendingFile && time.Now().Before(expireTime) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		if err := m.Files.Sync(ctx); err != nil {
			return err
		}
		syncedFile, err := m.Files.Get(file.ID)
		if err != nil {
			return err
		}
		*file = syncedFile
	}
	if file.Status != models.AvailableFile {
		return fmt.Errorf("file has invalid status: %s", file.Status)
	}
	return nil
}

func (m *FileManager) DownloadFile(ctx context.Context, id int64) (*os.File, error) {
	file, err := m.Files.Get(id)
	if err == sql.ErrNoRows {
		err = m.Files.Sync(ctx)
	}
	if err != nil {
		return nil, err
	}
	if err := m.waitFileAvailable(ctx, &file); err != nil {
		return nil, err
	}
	return os.Open(filepath.Join(m.Dir, filepath.FromSlash(file.Path)))
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
