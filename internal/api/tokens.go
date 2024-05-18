package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
)

// registerTokenHandlers registers handlers for token management.
func (v *View) registerTokenHandlers(g *echo.Group) {
	g.POST(
		"/v0/tokens/:token", v.consumeToken,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractToken,
		v.requirePermission(perms.ConsumeTokenRole),
	)
}

type consumeTokenForm struct {
	Secret string `json:"secret"`
}

type resetPasswordTokenForm struct {
	Password string `json:"password"`
}

func (v *View) consumeToken(c echo.Context) error {
	token, ok := c.Get(tokenKey).(models.Token)
	if !ok {
		c.Logger().Error("token not extracted")
		return fmt.Errorf("token not extracted")
	}
	if token.ExpireTime <= time.Now().Unix() {
		_ = v.core.Tokens.Delete(c.Request().Context(), token.ID)
		return errorResponse{
			Code:    http.StatusNotFound,
			Message: localize(c, "Invalid token ID."),
		}
	}
	form := consumeTokenForm{}
	if err := reusableBind(c, &form); err != nil {
		return err
	}
	if token.Secret != form.Secret {
		return errorResponse{
			Code:    http.StatusNotFound,
			Message: localize(c, "Invalid token ID."),
		}
	}
	switch token.Kind {
	case models.ConfirmEmailToken:
		var config models.ConfirmEmailTokenConfig
		if err := token.ScanConfig(&config); err != nil {
			return err
		}
		if err := v.core.WrapTx(c.Request().Context(), func(ctx context.Context) error {
			user, err := v.core.Users.Get(ctx, token.AccountID)
			if err != nil {
				return err
			}
			user.Email = models.NString(config.Email)
			if user.Status == models.PendingUser {
				user.Status = models.ActiveUser
			}
			if err := v.core.Users.Update(ctx, user); err != nil {
				return err
			}
			return v.core.Tokens.Delete(ctx, token.ID)
		}, sqlRepeatableRead); err != nil {
			return err
		}
	case models.ResetPasswordToken:
		var form resetPasswordTokenForm
		if err := c.Bind(&form); err != nil {
			return err
		}
		var errors errorFields
		validatePassword(c, errors, form.Password)
		if len(errors) > 0 {
			return errorResponse{
				Code:          http.StatusBadRequest,
				Message:       localize(c, "Form has invalid fields."),
				InvalidFields: errors,
			}
		}
		if err := v.core.WrapTx(c.Request().Context(), func(ctx context.Context) error {
			user, err := v.core.Users.Get(ctx, token.AccountID)
			if err != nil {
				return err
			}
			if err := v.core.Users.SetPassword(&user, form.Password); err != nil {
				return err
			}
			if err := v.core.Users.Update(ctx, user); err != nil {
				return err
			}
			return v.core.Tokens.Delete(ctx, token.ID)
		}, sqlRepeatableRead); err != nil {
			return err
		}
	default:
		return fmt.Errorf("token %v not supported", token.Kind)
	}
	return c.JSON(http.StatusOK, nil)
}

func (v *View) extractToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("token"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid token ID."),
			}
		}
		token, err := v.core.Tokens.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Invalid token ID."),
				}
			}
			c.Logger().Error(err)
			return err
		}
		c.Set(tokenKey, token)
		return next(c)
	}
}
