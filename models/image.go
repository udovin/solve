package models

import (
	"database/sql"
	"encoding/json"

	"github.com/udovin/gosql"
)

type ImageKind int

const (
	CompilerImage ImageKind = 1
)

type CompilerImageConfig struct {
}

// Image represents image.
type Image struct {
	ID      int64     `db:"id"`
	OwnerID NInt64    `db:"owner_id"`
	Name    string    `db:"name"`
	Kind    ImageKind `db:"kind"`
	Config  JSON      `db:"config"`
}

// ObjectID returns ID of image.
func (o Image) ObjectID() int64 {
	return o.ID
}

// Clone create copy of image.
func (o Image) Clone() Image {
	o.Config = o.Config.Clone()
	return o
}

func (o Image) ScanConfig(config any) error {
	return json.Unmarshal(o.Config, config)
}

func (o *Image) SetConfig(config any) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Config = raw
	return nil
}

// ImageEvent represents image event.
type ImageEvent struct {
	baseEvent
	Image
}

// Object returns event image.
func (e ImageEvent) Object() Image {
	return e.Image
}

// WithObject replaces event image.
func (e ImageEvent) WithObject(o Image) ObjectEvent[Image] {
	e.Image = o
	return e
}

// ImageStore represents store for images.
type ImageStore struct {
	baseStore[Image, ImageEvent]
	images map[int64]Image
}

// Get returns image by specified ID.
func (s *ImageStore) Get(id int64) (Image, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if image, ok := s.images[id]; ok {
		return image.Clone(), nil
	}
	return Image{}, sql.ErrNoRows
}

func (s *ImageStore) All() ([]Image, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var images []Image
	for _, image := range s.images {
		images = append(images, image)
	}
	return images, nil
}

func (s *ImageStore) reset() {
	s.images = map[int64]Image{}
}

func (s *ImageStore) makeObject(id int64) Image {
	return Image{ID: id}
}

func (s *ImageStore) makeObjectEvent(typ EventType) ObjectEvent[Image] {
	return ImageEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *ImageStore) onCreateObject(image Image) {
	s.images[image.ID] = image
}

func (s *ImageStore) onDeleteObject(image Image) {
	delete(s.images, image.ID)
}

func (s *ImageStore) onUpdateObject(image Image) {
	if old, ok := s.images[image.ID]; ok {
		s.onDeleteObject(old)
	}
	s.onCreateObject(image)
}

// NewImageStore creates a new instance of ImageStore.
func NewImageStore(db *gosql.DB, table, eventTable string) *ImageStore {
	impl := &ImageStore{}
	impl.baseStore = makeBaseStore[Image, ImageEvent](
		db, table, eventTable, impl,
	)
	return impl
}
