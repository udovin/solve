package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
)

// registerPostHandlers registers handlers for post management.
func (v *View) registerPostHandlers(g *echo.Group) {
	g.GET(
		"/v0/posts", v.observePosts,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(perms.ObservePostsRole),
	)
	g.POST(
		"/v0/posts", v.createPost,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(perms.CreatePostRole),
	)
	g.GET(
		"/v0/posts/:post", v.observePost,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractPost,
		v.requirePermission(perms.ObservePostRole),
	)
	g.PATCH(
		"/v0/posts/:post", v.updatePost,
		v.extractAuth(v.sessionAuth), v.extractPost,
		v.requirePermission(perms.UpdatePostRole),
	)
	g.DELETE(
		"/v0/posts/:post", v.deletePost,
		v.extractAuth(v.sessionAuth), v.extractPost,
		v.requirePermission(perms.DeletePostRole),
	)
	g.GET(
		"/v0/posts/:post/content/:name",
		v.observePostContent,
		v.extractAuth(v.sessionAuth), v.extractPost,
		v.requirePermission(perms.ObservePostRole),
	)
	g.GET(
		"/v0/users/:user/posts",
		v.observeUserPosts,
		v.extractAuth(v.sessionAuth), v.extractUser,
		v.requirePermission(perms.ObserveUserRole, perms.ObservePostsRole),
	)
}

type postsFilter struct {
	BeginID int64 `query:"begin_id"`
	Limit   int   `query:"limit"`
}

const (
	defaultPostLimit = 100
	maxPostLimit     = 5000
)

func (f *postsFilter) Parse(c echo.Context) error {
	if err := c.Bind(f); err != nil {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Invalid filter."),
		}
	}
	if f.BeginID < 0 || f.BeginID == math.MaxInt64 {
		f.BeginID = 0
	}
	if f.Limit <= 0 {
		f.Limit = defaultPostLimit
	}
	f.Limit = min(f.Limit, maxPostLimit)
	return nil
}

func (f *postsFilter) Filter(post models.Post) bool {
	if f.BeginID != 0 && post.ID > f.BeginID {
		return false
	}
	return true
}

func (v *View) observePosts(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		c.Logger().Error("auth not extracted")
		return fmt.Errorf("auth not extracted")
	}
	filter := postsFilter{Limit: 250}
	if err := filter.Parse(c); err != nil {
		c.Logger().Warn(err)
		return err
	}
	var resp Posts
	posts, err := v.core.Posts.ReverseAll(getContext(c), maxPostLimit+1, filter.BeginID)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	defer func() { _ = posts.Close() }()
	postsCount := 0
	for posts.Next() {
		post := posts.Row()
		if postsCount >= maxPostLimit ||
			len(resp.Posts) >= filter.Limit {
			resp.NextBeginID = post.ID
			break
		}
		postsCount++
		if !filter.Filter(post) {
			continue
		}
		permissions := v.getPostPermissions(accountCtx, post)
		if permissions.HasPermission(perms.ObservePostRole) {
			resp.Posts = append(resp.Posts, v.makePost(post, false))
		}
	}
	if err := posts.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observePost(c echo.Context) error {
	post, ok := c.Get(postKey).(models.Post)
	if !ok {
		c.Logger().Error("solution not extracted")
		return fmt.Errorf("solution not extracted")
	}
	return c.JSON(http.StatusOK, v.makePost(post, true))
}

type PostFormFile struct {
	Name    string      `json:"name"`
	Content *FileReader `json:"-"`
}

type CreatePostForm struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Files       []PostFormFile `json:"files"`
}

func (f *CreatePostForm) Parse(c echo.Context) error {
	var form struct {
		Data []byte `form:"data"`
	}
	if err := c.Bind(&form); err != nil {
		return err
	}
	if err := json.Unmarshal(form.Data, f); err != nil {
		return err
	}
	close := true
	defer func() {
		if close {
			f.Close()
		}
	}()
	for i := range f.Files {
		formFile, err := c.FormFile("file_" + f.Files[i].Name)
		if err != nil {
			return err
		}
		file, err := managers.NewMultipartFileReader(formFile)
		if err != nil {
			return err
		}
		f.Files[i].Content = file
	}
	close = false
	return nil
}

func (f *CreatePostForm) Close() {
	for i := range f.Files {
		if f.Files[i].Content != nil {
			_ = f.Files[i].Content.Close()
			f.Files[i].Content = nil
		}
	}
}

func (f *CreatePostForm) Update(c echo.Context, post *models.Post) error {
	errors := errorFields{}
	if len(f.Title) < 2 {
		errors["title"] = errorField{
			Message: localize(c, "Title is too short."),
		}
	} else if len(f.Title) > 64 {
		errors["title"] = errorField{
			Message: localize(c, "Title is too long."),
		}
	}
	post.Title = f.Title
	if len(f.Description) < 64 {
		errors["description"] = errorField{
			Message: localize(c, "Description is too short."),
		}
	} else if len(f.Description) > 65535 {
		errors["description"] = errorField{
			Message: localize(c, "Description is too long."),
		}
	}
	post.Description = f.Description
	if len(errors) > 0 {
		return &errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	return nil
}

func (v *View) createPost(c echo.Context) error {
	form := CreatePostForm{}
	if err := form.Parse(c); err != nil {
		return err
	}
	defer func() { form.Close() }()
	post := models.Post{}
	if err := form.Update(c, &post); err != nil {
		return err
	}
	if err := v.core.Posts.Create(getContext(c), &post); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, v.makePost(post, true))
}

type UpdatePostForm struct {
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	Files       []PostFormFile `json:"files"`
	DeleteFiles []int64        `json:"delete_files"`
}

func (f *UpdatePostForm) Parse(c echo.Context) error {
	var form struct {
		Data []byte `form:"data"`
	}
	if err := c.Bind(&form); err != nil {
		return err
	}
	if err := json.Unmarshal(form.Data, f); err != nil {
		return err
	}
	close := true
	defer func() {
		if close {
			f.Close()
		}
	}()
	for i := range f.Files {
		formFile, err := c.FormFile("file_" + f.Files[i].Name)
		if err != nil {
			return err
		}
		file, err := managers.NewMultipartFileReader(formFile)
		if err != nil {
			return err
		}
		f.Files[i].Content = file
	}
	close = false
	return nil
}

func (f *UpdatePostForm) Close() {
	for i := range f.Files {
		if f.Files[i].Content != nil {
			_ = f.Files[i].Content.Close()
			f.Files[i].Content = nil
		}
	}
}

func (f *UpdatePostForm) Update(c echo.Context, post *models.Post) error {
	errors := errorFields{}
	if f.Title != nil {
		if len(*f.Title) < 2 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too short."),
			}
		} else if len(*f.Title) > 64 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too long."),
			}
		}
		post.Title = *f.Title
	}
	if f.Description != nil {
		if len(*f.Description) < 64 {
			errors["description"] = errorField{
				Message: localize(c, "Description is too short."),
			}
		} else if len(*f.Description) > 65535 {
			errors["description"] = errorField{
				Message: localize(c, "Description is too long."),
			}
		}
		post.Description = *f.Description
	}
	if len(errors) > 0 {
		return &errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	return nil
}

func (v *View) updatePost(c echo.Context) error {
	post, ok := c.Get(postKey).(models.Post)
	if !ok {
		return fmt.Errorf("post not extracted")
	}
	form := UpdatePostForm{}
	if err := form.Parse(c); err != nil {
		return err
	}
	defer func() { form.Close() }()
	if err := form.Update(c, &post); err != nil {
		return err
	}
	if err := v.core.Posts.Update(getContext(c), post); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, v.makePost(post, true))
}

func (v *View) deletePost(c echo.Context) error {
	post, ok := c.Get(postKey).(models.Post)
	if !ok {
		return fmt.Errorf("post not extracted")
	}
	if err := v.core.Posts.Delete(getContext(c), post.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, v.makePost(post, false))
}

func (v *View) observePostContent(c echo.Context) error {
	return fmt.Errorf("not implemented")
}

func (v *View) observeUserPosts(c echo.Context) error {
	return fmt.Errorf("not implemented")
}

type Post struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type Posts struct {
	Posts       []Post `json:"posts"`
	NextBeginID int64  `json:"next_begin_id"`
}

func (v *View) makePost(post models.Post, withDescription bool) Post {
	resp := Post{
		ID:    post.ID,
		Title: post.Title,
	}
	if withDescription {
		resp.Description = post.Description
	}
	return resp
}

func (v *View) extractPost(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("post"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid post ID."),
			}
		}
		if err := syncStore(c, v.core.Posts); err != nil {
			return err
		}
		post, err := v.core.Posts.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Post not found."),
				}
			}
			return err
		}
		accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
		if !ok {
			return fmt.Errorf("account not extracted")
		}
		c.Set(postKey, post)
		c.Set(permissionCtxKey, v.getPostPermissions(accountCtx, post))
		return next(c)
	}
}

func (v *View) getPostPermissions(
	ctx *managers.AccountContext, post models.Post,
) perms.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if account := ctx.Account; account != nil &&
		post.OwnerID != 0 && account.ID == int64(post.OwnerID) {
		permissions.AddPermission(
			perms.ObservePostRole,
			perms.UpdatePostRole,
			perms.UpdatePostOwnerRole,
			perms.DeletePostRole,
		)
	} else if post.PublishTime != 0 &&
		permissions.HasPermission(perms.ObservePostsRole) {
		permissions.AddPermission(perms.ObservePostRole)
	}
	return permissions
}
