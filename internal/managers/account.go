package managers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/udovin/solve/internal/core"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
)

type AccountManager struct {
	accounts     *models.AccountStore
	users        *models.UserStore
	scopes       *models.ScopeStore
	scopeUsers   *models.ScopeUserStore
	groups       *models.GroupStore
	groupMembers *models.GroupMemberStore
	roles        *models.RoleStore
	roleEdges    *models.RoleEdgeStore
	accountRoles *models.AccountRoleStore
	settings     *models.SettingStore
}

func NewAccountManager(core *core.Core) *AccountManager {
	return &AccountManager{
		accounts:     core.Accounts,
		users:        core.Users,
		scopes:       core.Scopes,
		scopeUsers:   core.ScopeUsers,
		groups:       core.Groups,
		groupMembers: core.GroupMembers,
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
		Permissions: perms.PermissionSet{},
	}
	var roleIDs []int64
	if account != nil {
		switch account.Kind {
		case models.UserAccountKind:
			user, err := m.users.Get(ctx, account.ID)
			if err != nil {
				return nil, err
			}
			c.User = &user
			role, err := m.getUserRole(ctx, user.Status)
			if err != nil {
				return nil, err
			}
			roleIDs = append(roleIDs, role.ID)
			if err := func() error {
				// Blocked users cannot have additional permissions.
				if user.Status == models.BlockedUser {
					return nil
				}
				edges, err := m.accountRoles.FindByAccount(ctx, account.ID)
				if err != nil {
					return err
				}
				defer func() { _ = edges.Close() }()
				for edges.Next() {
					edge := edges.Row()
					roleIDs = append(roleIDs, edge.RoleID)
				}
				return edges.Err()
			}(); err != nil {
				return nil, err
			}
			if err := func() error {
				members, err := m.groupMembers.FindByAccount(ctx, account.ID)
				if err != nil {
					return err
				}
				defer func() { _ = members.Close() }()
				for members.Next() {
					member := members.Row()
					group, err := m.groups.Get(ctx, member.GroupID)
					if err != nil {
						return err
					}
					groupAccount, err := m.accounts.Get(ctx, group.ID)
					if err != nil {
						return err
					}
					c.GroupAccounts = append(c.GroupAccounts, groupAccount)
				}
				return members.Err()
			}(); err != nil {
				return nil, err
			}
		case models.ScopeUserAccountKind:
			user, err := m.scopeUsers.Get(ctx, account.ID)
			if err != nil {
				return nil, err
			}
			scope, err := m.scopes.Get(ctx, user.ScopeID)
			if err != nil {
				return nil, err
			}
			scopeAccount, err := m.accounts.Get(ctx, scope.ID)
			if err != nil {
				return nil, err
			}
			c.ScopeUser = &user
			c.GroupAccounts = append(c.GroupAccounts, scopeAccount)
			role, err := m.getScopeUserRole(ctx)
			if err != nil {
				return nil, err
			}
			roleIDs = append(roleIDs, role.ID)
		default:
			return nil, fmt.Errorf("unknown account kind: %v", account.Kind)
		}
	} else {
		role, err := m.getGuestRole(ctx)
		if err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, role.ID)
	}
	permissions, err := m.getRecursivePermissions(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	c.Permissions = permissions
	return &c, nil
}

func (m *AccountManager) getGuestRole(ctx context.Context) (models.Role, error) {
	roleName := "guest_group"
	roleNameSetting, err := m.settings.GetByKey("accounts.guest_role")
	if err == nil {
		roleName = roleNameSetting.Value
	} else if err != sql.ErrNoRows {
		return models.Role{}, err
	}
	return m.roles.GetByName(ctx, roleName)
}

func (m *AccountManager) getUserRole(ctx context.Context, status models.UserStatus) (models.Role, error) {
	roleName := fmt.Sprintf("%s_user_group", status)
	roleNameSetting, err := m.settings.GetByKey(fmt.Sprintf("accounts.%s_user_role", status))
	if err == nil {
		roleName = roleNameSetting.Value
	} else if err != sql.ErrNoRows {
		return models.Role{}, err
	}
	return m.roles.GetByName(ctx, roleName)
}

func (m *AccountManager) getScopeUserRole(ctx context.Context) (models.Role, error) {
	roleName := "scope_user_group"
	roleNameSetting, err := m.settings.GetByKey("accounts.scope_user_role")
	if err == nil {
		roleName = roleNameSetting.Value
	} else if err != sql.ErrNoRows {
		return models.Role{}, err
	}
	return m.roles.GetByName(ctx, roleName)
}

func (m *AccountManager) getRecursivePermissions(ctx context.Context, roleIDs []int64) (perms.PermissionSet, error) {
	roles := map[int64]struct{}{}
	for _, id := range roleIDs {
		roles[id] = struct{}{}
	}
	permissions := perms.PermissionSet{}
	for len(roleIDs) > 0 {
		for _, id := range roleIDs {
			role, err := m.roles.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			if perms.IsBuiltInRole(role.Name) {
				permissions.AddPermission(role.Name)
			}
		}
		if err := func() error {
			edges, err := m.roleEdges.FindByRole(ctx, roleIDs...)
			if err != nil {
				return err
			}
			defer func() { _ = edges.Close() }()
			// Clear previous roleIDs.
			roleIDs = roleIDs[:0]
			for edges.Next() {
				edge := edges.Row()
				if _, ok := roles[edge.ChildID]; !ok {
					roleIDs = append(roleIDs, edge.ChildID)
					roles[edge.ChildID] = struct{}{}
				}
			}
			if err := edges.Err(); err != nil {
				return err
			}
			return nil
		}(); err != nil {
			return nil, err
		}
	}
	return permissions, nil
}

type AccountContext struct {
	context       context.Context
	Account       *models.Account
	User          *models.User
	ScopeUser     *models.ScopeUser
	Permissions   perms.PermissionSet
	GroupAccounts []models.Account
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
	_ context.Context   = (*AccountContext)(nil)
	_ perms.Permissions = (*AccountContext)(nil)
)
