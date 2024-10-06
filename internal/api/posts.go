package api

import (
	"database/sql"
	"fmt"
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
		"/v0/posts/:post", v.deleteProblem,
		v.extractAuth(v.sessionAuth), v.extractPost,
		v.requirePermission(perms.DeletePostRole),
	)
	g.GET(
		"/v0/posts/:post/content/:name",
		v.observePostContent,
		v.extractAuth(v.sessionAuth), v.extractPost,
		v.requirePermission(perms.ObservePostRole),
	)
}

func (v *View) observePosts(c echo.Context) error {
	return fmt.Errorf("not implemented")
}

func (v *View) observePost(c echo.Context) error {
	return fmt.Errorf("not implemented")
}

func (v *View) createPost(c echo.Context) error {
	return fmt.Errorf("not implemented")
}

func (v *View) updatePost(c echo.Context) error {
	return fmt.Errorf("not implemented")
}

func (v *View) deletePost(c echo.Context) error {
	return fmt.Errorf("not implemented")
}

func (v *View) observePostContent(c echo.Context) error {
	return fmt.Errorf("not implemented")
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
