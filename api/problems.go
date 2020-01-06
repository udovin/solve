package api

import (
	"archive/zip"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

type Problem struct {
	models.Problem
	Title       string     `json:""`
	Description string     `json:""`
	Solutions   []Solution `json:",omitempty"`
}

func (v *View) CreateProblem(c echo.Context) error {
	var problem Problem
	if err := c.Bind(&problem); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	user, ok := c.Get(authUserKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !user.IsSuper {
		return c.NoContent(http.StatusForbidden)
	}
	problem.UserID = user.ID
	file, err := c.FormFile("File")
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	data, err := file.Open()
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	defer func() {
		if err := data.Close(); err != nil {
			c.Logger().Error(err)
		}
	}()
	tx, err := v.core.Problems.Manager.Begin()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := v.core.Problems.CreateTx(tx, &problem.Problem); err != nil {
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	pkg, err := os.Create(path.Join(
		v.core.Config.Invoker.ProblemsDir,
		fmt.Sprintf("%d.zip", problem.ID),
	))
	if err != nil {
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer func() {
		if err := pkg.Close(); err != nil {
			c.Logger().Error(err)
		}
	}()
	if _, err := io.Copy(pkg, data); err != nil {
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	description, err := v.extractPackageStatement(pkg.Name())
	if err != nil {
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	statement := models.Statement{
		ProblemID:   problem.ID,
		Title:       problem.Title,
		Description: description,
	}
	if err := v.core.Statements.CreateTx(tx, &statement); err != nil {
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := tx.Commit(); err != nil {
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, problem)
}

func (v *View) extractPackageStatement(path string) (string, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := archive.Close(); err != nil {
			log.Println("Error:", err)
		}
	}()
	for _, file := range archive.File {
		if file.Name == "statements/problem.html" {
			data, err := file.Open()
			if err != nil {
				return "", err
			}
			content, err := ioutil.ReadAll(data)
			if err != nil {
				if err := data.Close(); err != nil {
					log.Println("Error:", err)
				}
				return "", err
			}
			return string(content), nil
		}
	}
	return "", errors.New("unable to find problem statement")
}

func (v *View) GetProblem(c echo.Context) error {
	problemID, err := strconv.ParseInt(c.Param("ProblemID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	problem, err := v.buildProblem(problemID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, problem)
}

func (v *View) buildProblem(id int64) (Problem, error) {
	problem, err := v.core.Problems.Get(id)
	if err != nil {
		return Problem{}, err
	}
	statement, err := v.core.Statements.GetByProblem(problem.ID)
	if err != nil {
		return Problem{}, err
	}
	return Problem{
		Problem:     problem,
		Title:       statement.Title,
		Description: statement.Description,
	}, nil
}

func (v *View) UpdateProblem(c echo.Context) error {
	var problem Problem
	if err := c.Bind(&problem); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	user, ok := c.Get(authUserKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !user.IsSuper {
		return c.NoContent(http.StatusForbidden)
	}
	file, err := c.FormFile("File")
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	data, err := file.Open()
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	defer func() {
		if err := data.Close(); err != nil {
			c.Logger().Error(err)
		}
	}()
	_ = os.Remove(fmt.Sprintf("%d.zip", problem.ID))
	pkg, err := os.Create(path.Join(
		v.core.Config.Invoker.ProblemsDir,
		fmt.Sprintf("%d.zip", problem.ID),
	))
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer func() {
		if err := pkg.Close(); err != nil {
			c.Logger().Error(err)
		}
	}()
	if _, err := io.Copy(pkg, data); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	description, err := v.extractPackageStatement(pkg.Name())
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	statement := models.Statement{
		ProblemID:   problem.ID,
		Title:       problem.Title,
		Description: description,
	}
	if err := v.core.Statements.Create(&statement); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, problem)
}

func (v *View) DeleteProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}
