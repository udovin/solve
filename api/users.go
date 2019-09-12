package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

type User struct {
	models.User
	FirstName  string `json:",omitempty"`
	LastName   string `json:",omitempty"`
	MiddleName string `json:",omitempty"`
}

func (v *View) CreateUser(c echo.Context) error {
	var userData struct {
		Login      string `json:""`
		Email      string `json:""`
		Password   string `json:""`
		FirstName  string `json:""`
		LastName   string `json:""`
		MiddleName string `json:""`
	}
	if err := c.Bind(&userData); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusBadRequest)
	}
	user := models.User{
		Login: userData.Login,
	}
	if err := user.SetPassword(
		userData.Password, v.app.PasswordSalt,
	); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	tx, err := v.app.Users.Manager.Begin()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := func() error {
		if err := v.app.Users.CreateTx(tx, &user); err != nil {
			return err
		}
		email := models.UserField{
			Type: models.EmailField,
			Data: userData.Email,
		}
		if err := v.app.UserFields.CreateTx(tx, &email); err != nil {
			return err
		}
		if userData.FirstName != "" {
			field := models.UserField{
				Type: models.FirstNameField,
				Data: userData.FirstName,
			}
			if err := v.app.UserFields.CreateTx(tx, &field); err != nil {
				return err
			}
		}
		if userData.LastName != "" {
			field := models.UserField{
				Type: models.LastNameField,
				Data: userData.LastName,
			}
			if err := v.app.UserFields.CreateTx(tx, &field); err != nil {
				return err
			}
		}
		if userData.MiddleName != "" {
			field := models.UserField{
				Type: models.MiddleNameField,
				Data: userData.MiddleName,
			}
			if err := v.app.UserFields.CreateTx(tx, &field); err != nil {
				return err
			}
		}
		return tx.Commit()
	}(); err != nil {
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, user)
}

func (v *View) GetUser(c echo.Context) error {
	userID, err := strconv.ParseInt(c.Param("UserID"), 10, 64)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	user, ok := v.app.Users.Get(userID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	result := User{User: user}
	for _, field := range v.app.UserFields.GetByUser(userID) {
		switch field.Type {
		case models.FirstNameField:
			result.FirstName = field.Data
		case models.LastNameField:
			result.LastName = field.Data
		case models.MiddleNameField:
			result.MiddleName = field.Data
		}
	}
	return c.JSON(http.StatusOK, result)
}

func (v *View) UpdateUser(c echo.Context) error {
	userID, err := strconv.ParseInt(c.Param("UserID"), 10, 64)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	user, ok := v.app.Users.Get(userID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	var userData struct {
		Password *string `json:""`
	}
	if err := c.Bind(&userData); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if userData.Password != nil {
		if err := user.SetPassword(
			*userData.Password, v.app.PasswordSalt,
		); err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}
	if err := v.app.Users.Update(&user); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, user)
}

func (v *View) DeleteUser(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}
