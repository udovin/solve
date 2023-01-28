package managers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type AccountManager struct {
	accounts      *models.AccountStore
	users         *models.UserStore
	internalUsers *models.InternalUserStore
	roles         *models.RoleStore
	roleEdges     *models.RoleEdgeStore
	accountRoles  *models.AccountRoleStore
	settings      *models.SettingStore
}

func NewAccountManager(core *core.Core) *AccountManager {
	return &AccountManager{
		accounts:     core.Accounts,
		users:        core.Users,
		roles:        core.Roles,
		roleEdges:    core.RoleEdges,
		accountRoles: core.AccountRoles,
		settings:     core.Settings,
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
		switch account.Kind {
		case models.UserAccount:
			user, err := m.users.GetByAccount(account.ID)
			if err != nil {
				return nil, err
			}
			c.User = &user
			role, err := m.getUserRole(user.Status)
			if err != nil {
				return nil, err
			}
			roleIDs = append(roleIDs, role.ID)
			edges, err := m.accountRoles.FindByAccount(account.ID)
			if err != nil {
				return nil, err
			}
			for _, edge := range edges {
				roleIDs = append(roleIDs, edge.RoleID)
			}
		case models.InternalUserAccount:
			user, err := m.internalUsers.GetByAccount(account.ID)
			if err != nil {
				return nil, err
			}
			c.InternalUser = &user
			role, err := m.getInternalUserRole()
			if err != nil {
				return nil, err
			}
			roleIDs = append(roleIDs, role.ID)
		default:
			return nil, fmt.Errorf("unknown account kind: %v", account.Kind)
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
	roleNameSetting, err := m.settings.GetByKey("accounts.guest_role")
	if err == nil {
		roleName = roleNameSetting.Value
	} else if err != sql.ErrNoRows {
		return models.Role{}, err
	}
	return m.roles.GetByName(roleName)
}

func (m *AccountManager) getUserRole(status models.UserStatus) (models.Role, error) {
	roleName := fmt.Sprintf("%s_user_group", status)
	roleNameSetting, err := m.settings.GetByKey(fmt.Sprintf("accounts.%s_user_role", status))
	if err == nil {
		roleName = roleNameSetting.Value
	} else if err != sql.ErrNoRows {
		return models.Role{}, err
	}
	return m.roles.GetByName(roleName)
}

func (m *AccountManager) getInternalUserRole() (models.Role, error) {
	roleName := "internal_user_group"
	roleNameSetting, err := m.settings.GetByKey("accounts.internal_user_role")
	if err == nil {
		roleName = roleNameSetting.Value
	} else if err != sql.ErrNoRows {
		return models.Role{}, err
	}
	return m.roles.GetByName(roleName)
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
		role, err := m.roles.Get(roleID)
		if err != nil {
			return nil, err
		}
		if role.IsBuiltIn() {
			permissions[role.Name] = struct{}{}
		}
		edges, err := m.roleEdges.FindByRole(roleID)
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
	context      context.Context
	Account      *models.Account
	User         *models.User
	InternalUser *models.InternalUser
	Permissions  PermissionSet
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
