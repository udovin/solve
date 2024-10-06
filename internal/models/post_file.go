package models

import (
	"github.com/udovin/gosql"
)

// PostFile represents a post file.
type PostFile struct {
	baseObject
	PostID int64  `db:"post_id"`
	FileID int64  `db:"file_id"`
	Name   string `db:"name"`
}

// Clone creates copy of post file.
func (o PostFile) Clone() PostFile {
	return o
}

// PostFileEvent represents a post file event.
type PostFileEvent struct {
	baseEvent
	PostFile
}

// Object returns event post file.
func (e PostFileEvent) Object() PostFile {
	return e.PostFile
}

// SetObject sets event post file.
func (e *PostFileEvent) SetObject(o PostFile) {
	e.PostFile = o
}

type PostFileStore interface {
	Store[PostFile, PostFileEvent]
}

type cachedPostFileStore struct {
	cachedStore[PostFile, PostFileEvent, *PostFile, *PostFileEvent]
}

// NewCachedPostFileStore creates a new instance of PostFileStore.
func NewCachedPostFileStore(
	db *gosql.DB, table, eventTable string,
) PostFileStore {
	impl := &cachedPostFileStore{}
	impl.cachedStore = makeCachedStore[PostFile, PostFileEvent](
		db, table, eventTable, impl,
	)
	return impl
}
