package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

func (v *View) GetCompilers(c echo.Context) error {
	compilers, err := v.app.Compilers.All()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if compilers == nil {
		compilers = make([]models.Compiler, 0)
	}
	return c.JSON(http.StatusOK, compilers)
}

func (v *View) CreateCompiler(c echo.Context) error {
	var compiler models.Compiler
	if err := c.Bind(&compiler); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !user.IsSuper {
		return c.NoContent(http.StatusForbidden)
	}
	if err := v.app.Compilers.Create(&compiler); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, compiler)
}

func (v *View) GetCompiler(c echo.Context) error {
	compilerID, err := strconv.ParseInt(c.Param("ContestID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	compiler, err := v.app.Compilers.Get(compilerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, compiler)
}
