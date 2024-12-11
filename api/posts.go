package api

import (
	"context"

	"github.com/udovin/solve/api/schema"
)

type PostsClient interface {
	ObservePosts(ctx context.Context, r schema.ObservePostsRequest) (schema.ObservePostsResponse, error)
	ObservePost(ctx context.Context, r schema.ObservePostRequest) (schema.ObservePostResponse, error)
	CreatePost(ctx context.Context, r schema.CreatePostRequest) (schema.CreatePostResponse, error)
	UpdatePost(ctx context.Context, r schema.UpdatePostRequest) (schema.UpdatePostResponse, error)
	DeletePost(ctx context.Context, r schema.DeletePostRequest) (schema.DeletePostResponse, error)
}
