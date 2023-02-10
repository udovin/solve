package models

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/udovin/gosql"
)

type FileStatus int

const (
	PendingFile   FileStatus = 0
	AvailableFile FileStatus = 1
)

// String returns string representation.
func (t FileStatus) String() string {
	switch t {
	case PendingFile:
		return "pending"
	case AvailableFile:
		return "available"
	default:
		return fmt.Sprintf("FileStatus(%d)", t)
	}
}

type FileMeta struct {
	Name string `json:"name,omitempty"`
	Size int64  `json:"size"`
	MD5  string `json:"md5"`
}

// File represents a file.
type File struct {
	baseObject
	Status     FileStatus `db:"status"`
	ExpireTime NInt64     `db:"expire_time"`
	Path       string     `db:"path"`
	Meta       JSON       `db:"meta"`
}

// Clone creates copy of file.
func (o File) Clone() File {
	o.Meta = o.Meta.Clone()
	return o
}

func (o File) GetMeta() (FileMeta, error) {
	var config FileMeta
	if len(o.Meta) == 0 {
		return config, nil
	}
	err := json.Unmarshal(o.Meta, &config)
	return config, err
}

func (o *File) SetMeta(config FileMeta) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Meta = raw
	return nil
}

// FileEvent represents a file event.
type FileEvent struct {
	baseEvent
	File
}

// Object returns event file.
func (e FileEvent) Object() File {
	return e.File
}

// SetObject sets event file.
func (e *FileEvent) SetObject(o File) {
	e.File = o
}

type FileStore interface {
	Store[File]
	Base() FileStore
	Get(ctx context.Context, id int64) (File, error)
}

type fileStore struct {
	baseStore[File, FileEvent, *File, *FileEvent]
}

func (s *fileStore) Base() FileStore {
	return s
}

type cachedFileStore struct {
	fileStore
	cachedStore[File, FileEvent, *File, *FileEvent]
}

// newFileStore creates a new instance of FileStore.
func NewFileStore(
	db *gosql.DB, table, eventTable string,
) FileStore {
	impl := &fileStore{}
	impl.baseStore = makeBaseStore[File, FileEvent](db, table, eventTable)
	return impl
}

// NewCachedFileStore creates a new instance of FileStore.
func NewCachedFileStore(
	db *gosql.DB, table, eventTable string,
) FileStore {
	impl := &cachedFileStore{}
	impl.cachedStore = makeCachedStore[File, FileEvent](
		db, table, eventTable, impl,
	)
	return impl
}
