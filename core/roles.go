package core

import (
	"github.com/udovin/solve/models"
)

// Roles contains roles.
type Roles map[int64]struct{}

// GetGuestRoles returns roles for guest account.
func (c *Core) GetGuestRoles() (Roles, error) {
	role, err := c.Roles.GetByCode(models.GuestGroupRole)
	if err != nil {
		return Roles{}, err
	}
	return c.getRecursiveRoles(role.ID)
}

// GetUserRoles returns roles for empty user.
func (c *Core) GetUserRoles() (Roles, error) {
	role, err := c.Roles.GetByCode(models.UserGroupRole)
	if err != nil {
		return Roles{}, err
	}
	return c.getRecursiveRoles(role.ID)
}

// HasRole return true if role set has this role or parent role.
func (c *Core) HasRole(roles Roles, code string) (bool, error) {
	role, err := c.Roles.GetByCode(code)
	if err != nil {
		return false, err
	}
	_, ok := roles[role.ID]
	return ok, nil
}

// GetAccountRoles returns roles for account.
func (c *Core) GetAccountRoles(id int64) (Roles, error) {
	edges, err := c.AccountRoles.FindByAccount(id)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, edge := range edges {
		ids = append(ids, edge.RoleID)
	}
	return c.getRecursiveRoles(ids...)
}

// getRecursiveRoles returns recursive roles for specified list of roles.
func (c *Core) getRecursiveRoles(ids ...int64) (Roles, error) {
	stack, roles := ids, Roles{}
	for _, id := range stack {
		roles[id] = struct{}{}
	}
	for len(stack) > 0 {
		roleID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		edges, err := c.RoleEdges.FindByRole(roleID)
		if err != nil {
			return nil, err
		}
		for _, edge := range edges {
			role, err := c.Roles.Get(edge.ChildID)
			if err != nil {
				return nil, err
			}
			if _, ok := roles[role.ID]; !ok {
				roles[role.ID] = struct{}{}
				stack = append(stack, role.ID)
			}
		}
	}
	return roles, nil
}
