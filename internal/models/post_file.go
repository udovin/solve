package models

import (
	"context"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
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

	GetByPostName(ctx context.Context, postID int64, name string) (PostFile, error)
}

type cachedPostFileStore struct {
	cachedStore[PostFile, PostFileEvent, *PostFile, *PostFileEvent]
	byPost     *btreeIndex[int64, PostFile, *PostFile]
	byPostName *btreeIndex[pair[int64, string], PostFile, *PostFile]
}

func (s *cachedPostFileStore) FindByPost(
	ctx context.Context, postID ...int64,
) (db.Rows[PostFile], error) {
	s.mutex.RLock()
	return btreeIndexFind(
		s.byPost,
		s.objects.Iter(),
		s.mutex.RLocker(),
		postID,
		0,
	), nil
}

func (s *cachedPostFileStore) GetByPostName(
	ctx context.Context, postID int64, name string,
) (PostFile, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return btreeIndexGet(
		s.byPostName,
		s.objects.Iter(),
		makePair(postID, name),
	)
}

// NewCachedPostFileStore creates a new instance of PostFileStore.
func NewCachedPostFileStore(
	db *gosql.DB, table, eventTable string,
) PostFileStore {
	impl := &cachedPostFileStore{
		byPost: newBTreeIndex(
			func(o PostFile) (int64, bool) { return o.PostID, true },
			lessInt64,
		),
		byPostName: newBTreeIndex(
			func(o PostFile) (pair[int64, string], bool) {
				return makePair(o.PostID, o.Name), true
			},
			lessPairInt64String,
		),
	}
	impl.cachedStore = makeCachedStore[PostFile, PostFileEvent](
		db, table, eventTable, impl,
		// Indexes:
		impl.byPost, impl.byPostName,
	)
	return impl
}
