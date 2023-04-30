package managers

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type FileStorage interface {
	GeneratePath(context.Context) (string, error)
	ReadFile(context.Context, string) (io.ReadCloser, error)
	WriteFile(context.Context, string, io.Reader) (models.FileMeta, error)
	DeleteFile(context.Context, string) error
}

type LocalStorage struct {
	Dir string
}

func (s *LocalStorage) GeneratePath(ctx context.Context) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	filePath := path.Join(
		hex.EncodeToString(bytes[0:2]),
		hex.EncodeToString(bytes[2:]),
	)
	systemPath := filepath.Join(s.Dir, filepath.FromSlash(filePath))
	if _, err := os.Stat(systemPath); err == nil {
		return "", os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return filePath, nil
}

func (s *LocalStorage) ReadFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.Dir, filepath.FromSlash(filePath)))
}

func (s *LocalStorage) WriteFile(ctx context.Context, filePath string, file io.Reader) (models.FileMeta, error) {
	systemDir := filepath.Join(s.Dir, filepath.FromSlash(path.Dir(filePath)))
	if err := os.MkdirAll(systemDir, 0777); err != nil {
		return models.FileMeta{}, err
	}
	dst, err := os.Create(filepath.Join(s.Dir, filepath.FromSlash(filePath)))
	if err != nil {
		return models.FileMeta{}, err
	}
	defer func() { _ = dst.Close() }()
	reader := newStatsReader(file)
	size, err := io.Copy(dst, reader)
	if err != nil {
		return models.FileMeta{}, err
	}
	if size != reader.Size() {
		return models.FileMeta{}, fmt.Errorf(
			"invalid copy size: %d != %d",
			reader.Size(), size,
		)
	}
	if err := dst.Sync(); err != nil {
		return models.FileMeta{}, err
	}
	return models.FileMeta{Size: reader.Size(), MD5: reader.MD5()}, err
}

func (s *LocalStorage) DeleteFile(ctx context.Context, filePath string) error {
	path := filepath.Join(s.Dir, filepath.FromSlash(filePath))
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

type S3Storage struct {
	client     *s3.Client
	bucket     string
	pathPrefix string
}

func (s *S3Storage) GeneratePath(ctx context.Context) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	filePath := hex.EncodeToString(bytes)
	return filePath, nil
}

func (s *S3Storage) ReadFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.pathPrefix + filePath),
	})
	if err != nil {
		return nil, err
	}
	return object.Body, nil
}

type statsReader struct {
	reader io.Reader
	size   int64
	mutex  sync.Mutex
	md5    hash.Hash
}

func newStatsReader(r io.Reader) *statsReader {
	return &statsReader{
		reader: r,
		md5:    md5.New(),
	}
}

func (r *statsReader) Read(buf []byte) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	n, err := r.reader.Read(buf)
	if n > 0 {
		r.size += int64(n)
		_, _ = r.md5.Write(buf[:n])
	}
	return n, err
}

func (r *statsReader) Size() int64 {
	return r.size
}

func (r *statsReader) MD5() string {
	return hex.EncodeToString(r.md5.Sum(nil))
}

func (s *S3Storage) WriteFile(ctx context.Context, filePath string, file io.Reader) (models.FileMeta, error) {
	reader := newStatsReader(file)
	result, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.pathPrefix + filePath),
		Body:   reader,
	})
	if err != nil {
		return models.FileMeta{}, err
	}
	meta := models.FileMeta{Size: reader.Size(), MD5: reader.MD5()}
	if result.ETag == nil {
		return models.FileMeta{}, fmt.Errorf("empty checksum")
	}
	etag := strings.Trim(*result.ETag, "\"")
	if meta.MD5 != etag {
		return models.FileMeta{}, fmt.Errorf(
			"invalid checksum: %q != %q", meta.MD5, etag,
		)
	}
	return meta, err
}

func (s *S3Storage) DeleteFile(ctx context.Context, filePath string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.pathPrefix + filePath),
	})
	return err
}

type FileManager struct {
	files         models.FileStore
	storage       FileStorage
	uploadTimeout time.Duration
}

func NewFileManager(c *core.Core) *FileManager {
	var storage FileStorage
	switch t := c.Config.Storage.Options.(type) {
	case config.LocalStorageOptions:
		storage = &LocalStorage{
			Dir: t.FilesDir,
		}
	case config.S3StorageOptions:
		resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID && region == t.Region {
				return aws.Endpoint{
					PartitionID:   "yc",
					URL:           t.Endpoint,
					SigningRegion: t.Region,
				}, nil
			}
			return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
		})
		config := aws.Config{
			Region: t.Region,
			Credentials: credentials.NewStaticCredentialsProvider(
				t.AccessKeyID, t.SecretAccessKey, "",
			),
			EndpointResolverWithOptions: resolver,
		}
		storage = &S3Storage{
			client:     s3.NewFromConfig(config),
			bucket:     t.Bucket,
			pathPrefix: t.PathPrefix,
		}
	default:
		panic(fmt.Errorf(
			"driver %q is not supported",
			c.Config.Storage.Options.Driver(),
		))
	}
	return &FileManager{
		files:         c.Files,
		storage:       storage,
		uploadTimeout: 10 * time.Minute,
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

func NewFileReader(file *os.File) *FileReader {
	return &FileReader{Name: file.Name(), Reader: file}
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
func (m *FileManager) UploadFile(
	ctx context.Context, fileReader *FileReader,
) (models.File, error) {
	defer func() { _ = fileReader.Close() }()
	if tx := db.GetTx(ctx); tx != nil {
		return models.File{}, fmt.Errorf("cannot upload file in transaction")
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(m.uploadTimeout)
	}
	filePath, err := m.storage.GeneratePath(ctx)
	if err != nil {
		return models.File{}, fmt.Errorf("cannot generate path: %w", err)
	}
	file := models.File{
		Status:     models.PendingFile,
		ExpireTime: models.NInt64(deadline.Add(time.Minute).Unix()),
		Path:       filePath,
	}
	meta := models.FileMeta{
		Name: fileReader.Name,
		Size: fileReader.Size,
	}
	if err := file.SetMeta(meta); err != nil {
		return models.File{}, err
	}
	if err := m.files.Create(ctx, &file); err != nil {
		return models.File{}, err
	}
	stats, err := m.storage.WriteFile(ctx, filePath, fileReader.Reader)
	if err != nil {
		return models.File{}, err
	}
	meta.Size = stats.Size
	meta.MD5 = stats.MD5
	if err := file.SetMeta(meta); err != nil {
		return models.File{}, err
	}
	return file, nil
}

func (m *FileManager) ConfirmUploadFile(
	ctx context.Context, file *models.File,
) error {
	if file.Status != models.PendingFile {
		return fmt.Errorf("file shoud be in pending status")
	}
	clone := file.Clone()
	clone.Status = models.AvailableFile
	clone.ExpireTime = 0
	if err := m.files.Update(ctx, clone); err != nil {
		return err
	}
	*file = clone
	return nil
}

func (m *FileManager) DeleteFile(ctx context.Context, id int64) error {
	file, err := m.files.Get(ctx, id)
	if err != nil {
		return err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	expireTime := time.Unix(int64(file.ExpireTime), 0)
	if file.Status == models.PendingFile && time.Now().Before(expireTime) {
		return fmt.Errorf("cannot delete not uploaded file")
	}
	file.Status = models.PendingFile
	file.ExpireTime = models.NInt64(deadline.Unix())
	if err := m.files.Update(ctx, file); err != nil {
		return err
	}
	if err := m.storage.DeleteFile(ctx, file.Path); err != nil {
		return err
	}
	return m.files.Delete(ctx, file.ID)
}

func (m *FileManager) DownloadFile(
	ctx context.Context, id int64,
) (io.ReadCloser, error) {
	file, err := m.files.Get(models.WithSync(ctx), id)
	if err != nil {
		return nil, err
	}
	if err := m.waitFileAvailable(ctx, &file); err != nil {
		return nil, err
	}
	return m.storage.ReadFile(ctx, file.Path)
}

func (m *FileManager) waitFileAvailable(
	ctx context.Context, file *models.File,
) error {
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
		syncedFile, err := m.files.Get(models.WithSync(ctx), file.ID)
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
