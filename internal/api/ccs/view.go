package ccs

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/udovin/solve/internal/core"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
)

type View struct {
	core     *core.Core
	accounts *managers.AccountManager
	contests *managers.ContestManager
}

func (v *View) Register(g *echo.Group) {
	g.Use(middleware.Logger())
	g.GET(
		"/contests/:contest/event-feed", v.getEventFeed,
		v.extractAuth(v.basicAuth), v.extractContest,
	)
}

func NewView(core *core.Core) *View {
	return &View{
		core:     core,
		accounts: managers.NewAccountManager(core),
		contests: managers.NewContestManager(core),
	}
}

const (
	accountCtxKey = "account_ctx"
	contestCtxKey = "contest_ctx"
)

type authMethod func(c echo.Context) (bool, error)

func (v *View) extractAuth(authMethods ...authMethod) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for _, method := range authMethods {
				ok, err := method(c)
				if err != nil {
					return err
				}
				if ok {
					return next(c)
				}
			}
			return c.NoContent(http.StatusUnauthorized)
		}
	}
}

func (v *View) basicAuth(c echo.Context) (bool, error) {
	login, password, ok := c.Request().BasicAuth()
	if !ok {
		return false, nil
	}
	if login == "" || password == "" {
		return false, nil
	}
	user, err := v.core.Users.GetByLogin(c.Request().Context(), login)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if !v.core.Users.CheckPassword(user, password) {
		return false, nil
	}
	account, err := v.core.Accounts.Get(c.Request().Context(), user.ID)
	if err != nil {
		return false, err
	}
	if account.Kind != models.UserAccount {
		c.Logger().Errorf(
			"Account %v should have %v kind, but has %v",
			account.ID, models.UserAccount, account.Kind,
		)
		return false, fmt.Errorf("invalid account kind %q", account.Kind)
	}
	accountCtx, err := v.accounts.MakeContext(c.Request().Context(), &account)
	if err != nil {
		return false, err
	}
	c.Set(accountCtxKey, accountCtx)
	return true, nil
}

func (v *View) extractContest(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("contest"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return c.NoContent(http.StatusBadRequest)
		}
		contest, err := v.core.Contests.Get(c.Request().Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.NoContent(http.StatusNotFound)
			}
			c.Logger().Error(err)
			return err
		}
		accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
		if !ok {
			return fmt.Errorf("account not extracted")
		}
		contestCtx, err := v.contests.BuildContext(accountCtx, contest)
		if err != nil {
			return err
		}
		c.Set(contestCtxKey, contestCtx)
		return next(c)
	}
}
