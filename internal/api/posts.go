package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/api/schema"
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
	var resp schema.ObservePostsResponse
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
			resp.Posts = append(resp.Posts, v.makePost(post, permissions, false))
		}
	}
	if err := posts.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

type ObservePostRequest struct {
	ID        int64
	WithFiles bool `query:"with_files"`
}

func (v *View) observePost(c echo.Context) error {
	var request ObservePostRequest
	if err := c.Bind(&request); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	post, ok := c.Get(postKey).(models.Post)
	if !ok {
		c.Logger().Error("post not extracted")
		return fmt.Errorf("post not extracted")
	}
	permissions, ok := c.Get(permissionCtxKey).(perms.PermissionSet)
	if !ok {
		return fmt.Errorf("permissions not extracted")
	}
	resp := v.makePost(post, permissions, true)
	if request.WithFiles && permissions.HasPermission(perms.UpdatePostRole) {
		if err := syncStore(c, v.core.PostFiles); err != nil {
			return err
		}
		files, err := v.core.PostFiles.FindByPost(getContext(c), post.ID)
		if err != nil {
			return err
		}
		defer func() { _ = files.Close() }()
		for files.Next() {
			file := files.Row()
			resp.Files = append(resp.Files, schema.PostFile{
				ID:   file.ID,
				Name: file.Name,
			})
		}
		if err := files.Err(); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, resp)
}

type PostFormFile struct {
	Name    string      `json:"name"`
	Content *FileReader `json:"-"`
}

type CreatePostForm struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Files       []PostFormFile `json:"files"`
	Publish     bool           `json:"publish"`
}

func (f *CreatePostForm) Parse(c echo.Context) error {
	var form struct {
		Data string `form:"data"`
	}
	if err := c.Bind(&form); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(form.Data), f); err != nil {
		return err
	}
	close := true
	defer func() {
		if close {
			f.Close()
		}
	}()
	uploadedFiles := map[string]struct{}{}
	for i := range f.Files {
		if _, ok := uploadedFiles[f.Files[i].Name]; ok {
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Form has invalid fields."),
				InvalidFields: errorFields{
					"files": {
						Message: localize(c, "Form has invalid fields."),
					},
				},
			}
		}
		uploadedFiles[f.Files[i].Name] = struct{}{}
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
	if len(f.Description) < 16 {
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
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	form := CreatePostForm{}
	if err := form.Parse(c); err != nil {
		return err
	}
	defer func() { form.Close() }()
	post := models.Post{}
	if err := form.Update(c, &post); err != nil {
		return err
	}
	now := getNow(c)
	post.CreateTime = now.Unix()
	if form.Publish {
		post.PublishTime = models.NInt64(now.Unix())
	}
	if account := accountCtx.Account; account != nil {
		post.OwnerID = NInt64(account.ID)
	}
	var files []models.File
	for _, f := range form.Files {
		file, err := v.files.UploadFile(getContext(c), f.Content)
		if err != nil {
			return err
		}
		files = append(files, file)
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		if err := v.core.Posts.Create(ctx, &post); err != nil {
			return err
		}
		for i, file := range files {
			if err := v.files.ConfirmUploadFile(ctx, &file); err != nil {
				return err
			}
			postFile := models.PostFile{
				PostID: post.ID,
				FileID: file.ID,
				Name:   form.Files[i].Name,
			}
			if err := v.core.PostFiles.Create(ctx, &postFile); err != nil {
				return err
			}
		}
		return nil
	}, sqlRepeatableRead); err != nil {
		return err
	}
	permissions := v.getPostPermissions(accountCtx, post)
	return c.JSON(http.StatusCreated, v.makePost(post, permissions, true))
}

type UpdatePostForm struct {
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	Publish     *bool          `json:"publish"`
	Files       []PostFormFile `json:"files"`
	DeleteFiles []int64        `json:"delete_files"`
	OwnerID     *int64         `json:"owner_id"`
}

func (f *UpdatePostForm) Parse(c echo.Context) error {
	var form struct {
		Data string `form:"data"`
	}
	if err := c.Bind(&form); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(form.Data), f); err != nil {
		return err
	}
	close := true
	defer func() {
		if close {
			f.Close()
		}
	}()
	uploadFiles := map[string]struct{}{}
	for i := range f.Files {
		if _, ok := uploadFiles[f.Files[i].Name]; ok {
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Form has invalid fields."),
				InvalidFields: errorFields{
					"files": {
						Message: localize(c, "Form has invalid fields."),
					},
				},
			}
		}
		uploadFiles[f.Files[i].Name] = struct{}{}
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
		if len(*f.Description) < 16 {
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
	permissions, ok := c.Get(permissionCtxKey).(perms.PermissionSet)
	if !ok {
		return fmt.Errorf("permissions not extracted")
	}
	form := UpdatePostForm{}
	if err := form.Parse(c); err != nil {
		return err
	}
	defer func() { form.Close() }()
	if err := form.Update(c, &post); err != nil {
		return err
	}
	now := getNow(c)
	if form.Publish != nil {
		if !*form.Publish {
			post.PublishTime = 0
		} else if post.PublishTime == 0 {
			post.PublishTime = models.NInt64(now.Unix())
		}
	}
	ctx := getContext(c)
	var missingPermissions []string
	if form.OwnerID != nil {
		if !permissions.HasPermission(perms.UpdateProblemOwnerRole) {
			missingPermissions = append(missingPermissions, perms.UpdateProblemOwnerRole)
		} else {
			account, err := v.core.Accounts.Get(ctx, *form.OwnerID)
			if err != nil {
				if err == sql.ErrNoRows {
					return errorResponse{
						Code:    http.StatusBadRequest,
						Message: localize(c, "User not found."),
					}
				}
				return err
			}
			if account.Kind != models.UserAccountKind {
				return errorResponse{
					Code:    http.StatusBadRequest,
					Message: localize(c, "User not found."),
				}
			}
			post.OwnerID = models.NInt64(*form.OwnerID)
		}
	}
	if len(missingPermissions) > 0 {
		return errorResponse{
			Code:               http.StatusForbidden,
			Message:            localize(c, "Account missing permissions."),
			MissingPermissions: missingPermissions,
		}
	}
	deleteFiles := map[string]struct{}{}
	for _, id := range form.DeleteFiles {
		file, err := v.core.PostFiles.Get(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusBadRequest,
					Message: localize(c, "Form has invalid fields."),
					InvalidFields: errorFields{
						"delete_files": {
							Message: localize(c, "Form has invalid fields."),
						},
					},
				}
			}
			return err
		}
		if file.PostID != post.ID {
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Form has invalid fields."),
				InvalidFields: errorFields{
					"delete_files": {
						Message: localize(c, "Form has invalid fields."),
					},
				},
			}
		}
		if _, ok := deleteFiles[file.Name]; ok {
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Form has invalid fields."),
				InvalidFields: errorFields{
					"delete_files": {
						Message: localize(c, "Form has invalid fields."),
					},
				},
			}
		}
		deleteFiles[file.Name] = struct{}{}
	}
	for _, f := range form.Files {
		if _, ok := deleteFiles[f.Name]; ok {
			continue
		}
		_, err := v.core.PostFiles.GetByPostName(ctx, post.ID, f.Name)
		if err != sql.ErrNoRows {
			if err == nil {
				return errorResponse{
					Code:    http.StatusBadRequest,
					Message: localize(c, "Form has invalid fields."),
					InvalidFields: errorFields{
						"files": {
							Message: localize(c, "Form has invalid fields."),
						},
					},
				}
			}
			return err
		}
	}
	var files []models.File
	for _, f := range form.Files {
		file, err := v.files.UploadFile(ctx, f.Content)
		if err != nil {
			return err
		}
		files = append(files, file)
	}
	if err := v.core.WrapTx(ctx, func(ctx context.Context) error {
		if err := v.core.Posts.Update(ctx, post); err != nil {
			return err
		}
		for _, id := range form.DeleteFiles {
			if err := v.core.PostFiles.Delete(ctx, id); err != nil {
				return err
			}
		}
		for i, file := range files {
			if err := v.files.ConfirmUploadFile(ctx, &file); err != nil {
				return err
			}
			postFile := models.PostFile{
				PostID: post.ID,
				FileID: file.ID,
				Name:   form.Files[i].Name,
			}
			if err := v.core.PostFiles.Create(ctx, &postFile); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, v.makePost(post, permissions, true))
}

func (v *View) deletePost(c echo.Context) error {
	post, ok := c.Get(postKey).(models.Post)
	if !ok {
		return fmt.Errorf("post not extracted")
	}
	permissions, ok := c.Get(permissionCtxKey).(perms.PermissionSet)
	if !ok {
		return fmt.Errorf("permissions not extracted")
	}
	if err := v.core.Posts.Delete(getContext(c), post.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, v.makePost(post, permissions, false))
}

func (v *View) observePostContent(c echo.Context) error {
	post, ok := c.Get(postKey).(models.Post)
	if !ok {
		return fmt.Errorf("problem not extracted")
	}
	if err := syncStore(c, v.core.PostFiles); err != nil {
		return err
	}
	if err := syncStore(c, v.core.Files); err != nil {
		return err
	}
	resourceName := c.Param("name")
	ctx := getContext(c)
	postFile, err := v.core.PostFiles.GetByPostName(ctx, post.ID, resourceName)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResponse{
				Code:    http.StatusNotFound,
				Message: localize(c, "File not found."),
			}
		}
		return err
	}
	file, err := v.core.Files.Get(getContext(c), int64(postFile.FileID))
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResponse{
				Code:    http.StatusNotFound,
				Message: localize(c, "File not found."),
			}
		}
		return err
	}
	c.Set(fileKey, file)
	return v.observeFileContent(c)
}

func (v *View) observeUserPosts(c echo.Context) error {
	return fmt.Errorf("not implemented")
}

var postPermissions = []string{
	perms.UpdatePostRole,
	perms.UpdatePostOwnerRole,
	perms.DeletePostRole,
}

func (v *View) makePost(post models.Post, permissions perms.Permissions, withDescription bool) schema.Post {
	resp := schema.Post{
		ID:          post.ID,
		Title:       post.Title,
		CreateTime:  post.CreateTime,
		PublishTime: int64(post.PublishTime),
	}
	if withDescription {
		resp.Description = post.Description
	}
	for _, permission := range postPermissions {
		if permissions.HasPermission(permission) {
			resp.Permissions = append(resp.Permissions, permission)
		}
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
			perms.DeletePostRole,
		)
	} else if post.PublishTime != 0 &&
		permissions.HasPermission(perms.ObservePostsRole) {
		permissions.AddPermission(perms.ObservePostRole)
	}
	return permissions
}
