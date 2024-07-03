package api

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
	"github.com/udovin/solve/internal/pkg/logs"
)

func (v *View) registerAccountHandlers(g *echo.Group) {
	g.GET(
		"/v0/accounts", v.observeAccounts,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(perms.ObserveAccountsRole),
	)
}

const maxAccountLimit = 5000

type accountFilter struct {
	Kind    string `query:"kind"`
	Query   string `query:"q"`
	BeginID int64  `query:"begin_id"`
	Limit   int    `query:"limit"`
}

func (f *accountFilter) Parse(c echo.Context) error {
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
		f.Limit = maxAccountLimit
	}
	f.Limit = min(f.Limit, maxAccountLimit)
	return nil
}

type Account struct {
	ID        int64      `json:"id"`
	Kind      string     `json:"kind"`
	User      *User      `json:"user,omitempty"`
	ScopeUser *ScopeUser `json:"scope_user,omitempty"`
	Scope     *Scope     `json:"scope,omitempty"`
}

type Accounts struct {
	Accounts    []Account `json:"accounts"`
	NextBeginID int64     `json:"next_begin_id,omitempty"`
}

func (f *accountFilter) Filter(account models.Account) bool {
	if f.BeginID != 0 && account.ID > f.BeginID {
		return false
	}
	if f.Kind != "" {
		switch account.Kind {
		case models.UserAccountKind:
			if f.Kind != "user" {
				return false
			}
		case models.ScopeUserAccountKind:
			if f.Kind != "scope_user" {
				return false
			}
		case models.ScopeAccountKind:
			if f.Kind != "scope" {
				return false
			}
		case models.GroupAccountKind:
			if f.Kind != "group" {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (f *accountFilter) FilterUser(user models.User) bool {
	if f.Query != "" {
		if !strings.Contains(strings.ToLower(user.Login), strings.ToLower(f.Query)) {
			return false
		}
	}
	return true
}

func (f *accountFilter) FilterScopeUser(user models.ScopeUser) bool {
	if f.Query != "" {
		if !strings.Contains(strings.ToLower(user.Login), strings.ToLower(f.Query)) {
			return false
		}
	}
	return true
}

func (f *accountFilter) FilterScope(scope models.Scope) bool {
	if f.Query != "" {
		if !strings.Contains(strings.ToLower(scope.Title), strings.ToLower(f.Query)) {
			return false
		}
	}
	return true
}

func (f *accountFilter) FilterGroup(group models.Group) bool {
	if f.Query != "" {
		if !strings.Contains(strings.ToLower(group.Title), strings.ToLower(f.Query)) {
			return false
		}
	}
	return true
}

func (v *View) observeAccounts(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	var filter accountFilter
	if err := filter.Parse(c); err != nil {
		c.Logger().Warn(err)
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Form has invalid fields."),
		}
	}
	var resp Accounts
	ctx := getContext(c)
	accounts, err := v.core.Accounts.ReverseAll(ctx, filter.Limit+1, filter.BeginID)
	if err != nil {
		return err
	}
	defer func() { _ = accounts.Close() }()
	accountsCount := 0
	for accounts.Next() {
		account := accounts.Row()
		if accountsCount >= filter.Limit {
			resp.NextBeginID = account.ID
			break
		}
		accountsCount++
		if !filter.Filter(account) {
			continue
		}
		switch account.Kind {
		case models.UserAccountKind:
			user, err := v.core.Users.Get(ctx, account.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					c.Logger().Warn("Cannot find user", logs.Any("id", account.ID))
					continue
				}
				return err
			}
			if !filter.FilterUser(user) {
				continue
			}
			permissions := v.getUserPermissions(accountCtx, user)
			if permissions.HasPermission(perms.ObserveUserRole) {
				userResp := makeUser(user, permissions)
				resp.Accounts = append(resp.Accounts, Account{
					ID:   account.ID,
					Kind: "user",
					User: &userResp,
				})
			}
		case models.ScopeUserAccountKind:
			scopeUser, err := v.core.ScopeUsers.Get(ctx, account.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					c.Logger().Warn("Cannot find scope user", logs.Any("id", account.ID))
					continue
				}
				return err
			}
			if !filter.FilterScopeUser(scopeUser) {
				continue
			}
			scope, err := v.core.Scopes.Get(ctx, scopeUser.ScopeID)
			if err != nil {
				if err == sql.ErrNoRows {
					c.Logger().Warn("Cannot find user scope", logs.Any("id", scopeUser.ScopeID))
					continue
				}
				return err
			}
			permissions := v.getScopePermissions(accountCtx, scope)
			if permissions.HasPermission(perms.ObserveScopeUserRole) {
				scopeUserResp := makeScopeUser(scopeUser)
				resp.Accounts = append(resp.Accounts, Account{
					ID:        account.ID,
					Kind:      "scope_user",
					ScopeUser: &scopeUserResp,
				})
			}
		case models.ScopeAccountKind:
			scope, err := v.core.Scopes.Get(ctx, account.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					c.Logger().Warn("Cannot find scope", logs.Any("id", account.ID))
					continue
				}
				return err
			}
			if !filter.FilterScope(scope) {
				continue
			}
			permissions := v.getScopePermissions(accountCtx, scope)
			if !permissions.HasPermission(perms.ObserveScopeRole) {
				scopeResp := makeScope(scope)
				resp.Accounts = append(resp.Accounts, Account{
					ID:    account.ID,
					Kind:  "scope",
					Scope: &scopeResp,
				})
			}
		case models.GroupAccountKind:
			group, err := v.core.Groups.Get(ctx, account.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					c.Logger().Warn("Cannot find group", logs.Any("id", account.ID))
					continue
				}
				return err
			}
			// TODO.
			_ = group
		default:
			c.Logger().Warn(
				"Unsupported account kind",
				logs.Any("id", account.ID),
				logs.Any("kind", account.Kind),
			)
		}
	}
	if err := accounts.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}
