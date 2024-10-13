package models

import (
	"github.com/udovin/gosql"
)

// Post represents a post.
type Post struct {
	baseObject
	OwnerID     int64  `db:"owner_id"`
	Title       string `db:"title"`
	Description string `db:"description"`
	CreateTime  int64  `db:"create_time"`
	PublishTime NInt64 `db:"publish_time"`
}

// Clone creates copy of post.
func (o Post) Clone() Post {
	return o
}

// PostEvent represents a post event.
type PostEvent struct {
	baseEvent
	Post
}

// Object returns event post.
func (e PostEvent) Object() Post {
	return e.Post
}

// SetObject sets event post.
func (e *PostEvent) SetObject(o Post) {
	e.Post = o
}

type PostStore interface {
	Store[Post, PostEvent]
}

type cachedPostStore struct {
	cachedStore[Post, PostEvent, *Post, *PostEvent]
}

// NewCachedFileStore creates a new instance of PostStore.
func NewCachedPostStore(
	db *gosql.DB, table, eventTable string,
) PostStore {
	impl := &cachedPostStore{}
	impl.cachedStore = makeCachedStore[Post, PostEvent](
		db, table, eventTable, impl,
	)
	return impl
}
