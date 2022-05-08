package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

func (v *View) registerImageHandlers(g *echo.Group) {
	g.GET(
		"/v0/images", v.ObserveImages,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(models.ObserveImagesRole),
	)
	g.POST(
		"/v0/image", v.createImage,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.CreateImageRole),
	)
	g.PATCH(
		"/v0/image/:image", v.updateImage,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.UpdateImageRole),
	)
	g.DELETE(
		"/v0/image/:image", v.deleteImage,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.DeleteImageRole),
	)
}

type Image struct {
	ID     int64            `json:"id"`
	Name   string           `json:"name"`
	Kind   models.ImageKind `json:"kind"`
	Config models.JSON      `json:"config"`
}

type Images struct {
	Images []Image `json:"images"`
}

type imageSorter []Image

func (v imageSorter) Len() int {
	return len(v)
}

func (v imageSorter) Less(i, j int) bool {
	return v[i].ID > v[j].ID
}

func (v imageSorter) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// ObserveImages returns list of available images.
func (v *View) ObserveImages(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	images, err := v.core.Images.All()
	if err != nil {
		return err
	}
	var resp Images
	for _, image := range images {
		permissions := v.getImagePermissions(accountCtx, image)
		if permissions.HasPermission(models.ObserveImageRole) {
			resp.Images = append(resp.Images, makeImage(image))
		}
	}
	sort.Sort(imageSorter(resp.Images))
	return c.JSON(http.StatusOK, resp)
}

type createImageForm struct {
	Name   string           `form:"name" json:"name"`
	Kind   models.ImageKind `form:"kind" json:"kind"`
	Config models.JSON      `form:"config" json:"config"`
}

func (f createImageForm) Update(image *models.Image) *errorResponse {
	errors := errorFields{}
	if len(f.Name) < 4 {
		errors["name"] = errorField{Message: "name is too short"}
	}
	if len(f.Name) > 64 {
		errors["name"] = errorField{Message: "name is too long"}
	}
	if len(errors) > 0 {
		return &errorResponse{
			Code:          http.StatusBadRequest,
			Message:       "form has invalid fields",
			InvalidFields: errors,
		}
	}
	image.Name = f.Name
	image.Kind = f.Kind
	image.Config = f.Config
	return nil
}

func (v *View) createImage(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	var form createImageForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var image models.Image
	if err := form.Update(&image); err != nil {
		return err
	}
	if account := accountCtx.Account; account != nil {
		image.OwnerID = models.NInt64(account.ID)
	}
	if err := v.core.WrapTx(c.Request().Context(), func(ctx context.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer func() {
			_ = src.Close()
		}()
		if err := v.core.Images.Create(ctx, &image); err != nil {
			return err
		}
		dst, err := os.Create(filepath.Join(
			v.core.Config.Storage.ImagesDir,
			fmt.Sprintf("%d.zip", image.ID),
		))
		if err != nil {
			return err
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		return err
	}, sqlRepeatableRead); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeImage(image))
}

func (v *View) updateImage(c echo.Context) error {
	return errNotImplemented
}

func (v *View) deleteImage(c echo.Context) error {
	image, ok := c.Get(imageKey).(models.Image)
	if !ok {
		return fmt.Errorf("image not extracted")
	}
	if err := v.core.Images.Delete(c.Request().Context(), image.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, makeImage(image))
}

func makeImage(image models.Image) Image {
	return Image{
		ID:     image.ID,
		Name:   image.Name,
		Kind:   image.Kind,
		Config: image.Config,
	}
}

func (v *View) getImagePermissions(
	ctx *managers.AccountContext, image models.Image,
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	permissions[models.ObserveImageRole] = struct{}{}
	return permissions
}
