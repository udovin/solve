package api

import (
	"database/sql"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/models"
)

func (v *View) registerFileHandlers(g *echo.Group) {
	g.GET(
		"/v0/files/:file/content/*", v.observeFileContent,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractFile,
		v.requirePermission(models.ObserveFileContentRole),
	)
}

func (v *View) observeFileContent(c echo.Context) error {
	file, ok := c.Get(fileKey).(models.File)
	if !ok {
		return fmt.Errorf("file not extracted")
	}
	meta, err := file.GetMeta()
	if err != nil {
		return err
	}
	content, err := v.files.DownloadFile(c.Request().Context(), file.ID)
	if err != nil {
		return err
	}
	contentType := mime.TypeByExtension(filepath.Ext(meta.Name))
	return c.Stream(http.StatusOK, contentType, content)
}

func (v *View) extractFile(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("file"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid file ID."),
			}
		}
		if err := syncStore(c, v.core.Files); err != nil {
			return err
		}
		file, err := v.core.Files.Get(id)
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
		return next(c)
	}
}
