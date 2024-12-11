package schema

type PostFile struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Post struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	CreateTime  int64      `json:"create_time,omitempty"`
	PublishTime int64      `json:"publish_time,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
	Files       []PostFile `json:"files,omitempty"`
}

type ObservePostsRequest struct {
}

type ObservePostsResponse struct {
	Posts       []Post `json:"posts"`
	NextBeginID int64  `json:"next_begin_id"`
}

type ObservePostRequest struct {
	ID        int64 `json:"-"`
	WithFiles bool  `json:"-"`
}

type ObservePostResponse struct {
	Post
}

type CreatePostRequestFile struct {
	Name string `json:"name"`
	File any    `json:"-"`
}

type CreatePostRequest struct {
	Title       string
	Description string
	Publish     bool
	Files       []CreatePostRequestFile
}

type CreatePostResponse struct {
	Post
}

type UpdatePostRequestFile = CreatePostRequestFile

type UpdatePostRequest struct {
	ID          int64                   `json:"-"`
	Title       *string                 `json:"title"`
	Description *string                 `json:"description"`
	Publish     *bool                   `json:"publish"`
	Files       []UpdatePostRequestFile `json:"files"`
	DeleteFiles []int64                 `json:"delete_files"`
}

type UpdatePostResponse struct {
	Post
}

type DeletePostRequest struct {
	ID int64 `json:"-"`
}

type DeletePostResponse struct {
	Post
}
