package managers

import (
	"context"
	"database/sql"
	"time"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type AccountManager struct {
	Accounts     *models.AccountStore
	Users        *models.UserStore
	Roles        *models.RoleStore
	RoleEdges    *models.RoleEdgeStore
	AccountRoles *models.AccountRoleStore
	Settings     *models.SettingStore
}

func NewAccountManager(core *core.Core) *AccountManager {
	return &AccountManager{
		Accounts:     core.Accounts,
		Users:        core.Users,
		Roles:        core.Roles,
		RoleEdges:    core.RoleEdges,
		AccountRoles: core.AccountRoles,
		Settings:     core.Settings,
	}
}

func (m *AccountManager) MakeContext(ctx context.Context, account *models.Account) (*AccountContext, error) {
	c := AccountContext{
		context:     ctx,
		Account:     account,
		Permissions: PermissionSet{},
	}
	var roleIDs []int64
	if account != nil {
		if account.Kind == models.UserAccount {
			user, err := m.Users.GetByAccount(account.ID)
			if err != nil {
				return nil, err
			}
			c.User = &user
			role, err := m.getUserRole()
			if err != nil {
				return nil, err
			}
			roleIDs = append(roleIDs, role.ID)
		}
		edges, err := m.AccountRoles.FindByAccount(account.ID)
		if err != nil {
			return nil, err
		}
		for _, edge := range edges {
			roleIDs = append(roleIDs, edge.RoleID)
		}
	} else {
		role, err := m.getGuestRole()
		if err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, role.ID)
	}
	permissions, err := m.getRecursivePermissions(roleIDs...)
	if err != nil {
		return nil, err
	}
	c.Permissions = permissions
	return &c, nil
}

func (m *AccountManager) getGuestRole() (models.Role, error) {
	roleName := "guest_group"
	roleNameSetting, err := m.Settings.GetByKey("accounts.guest_role")
	if err == nil {
		roleName = roleNameSetting.Value
	} else if err != sql.ErrNoRows {
		return models.Role{}, err
	}
	return m.Roles.GetByName(roleName)
}

func (m *AccountManager) getUserRole() (models.Role, error) {
	roleName := "user_group"
	roleNameSetting, err := m.Settings.GetByKey("accounts.user_role")
	if err == nil {
		roleName = roleNameSetting.Value
	} else if err != sql.ErrNoRows {
		return models.Role{}, err
	}
	return m.Roles.GetByName(roleName)
}

func (m *AccountManager) getRecursivePermissions(roleIDs ...int64) (PermissionSet, error) {
	roles := map[int64]struct{}{}
	for _, id := range roleIDs {
		roles[id] = struct{}{}
	}
	stack, permissions := roleIDs, PermissionSet{}
	for len(stack) > 0 {
		roleID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		role, err := m.Roles.Get(roleID)
		if err != nil {
			return nil, err
		}
		if role.IsBuiltIn() {
			permissions[role.Name] = struct{}{}
		}
		edges, err := m.RoleEdges.FindByRole(roleID)
		if err != nil {
			return nil, err
		}
		for _, edge := range edges {
			if _, ok := roles[edge.ChildID]; !ok {
				stack = append(stack, edge.ChildID)
				roles[edge.ChildID] = struct{}{}
			}
		}
	}
	return permissions, nil
}

type Permissions interface {
	HasPermission(name string) bool
}

type PermissionSet map[string]struct{}

func (p PermissionSet) AddPermission(names ...string) {
	for _, name := range names {
		p[name] = struct{}{}
	}
}

func (p PermissionSet) HasPermission(name string) bool {
	_, ok := p[name]
	return ok
}

func (p PermissionSet) Clone() PermissionSet {
	clone := PermissionSet{}
	for key := range p {
		clone[key] = struct{}{}
	}
	return clone
}

type AccountContext struct {
	context     context.Context
	Account     *models.Account
	User        *models.User
	Permissions PermissionSet
}

func (c *AccountContext) HasPermission(name string) bool {
	return c.Permissions.HasPermission(name)
}

func (c *AccountContext) Deadline() (time.Time, bool) {
	return c.context.Deadline()
}

func (c *AccountContext) Done() <-chan struct{} {
	return c.context.Done()
}

func (c *AccountContext) Err() error {
	return c.context.Err()
}

func (c *AccountContext) Value(key any) any {
	return c.context.Value(key)
}

var (
	_ context.Context = (*AccountContext)(nil)
	_ Permissions     = (*AccountContext)(nil)
)
