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
	return c.getGroupRoles(role.ID)
}

// GetUserRoles returns roles for user.
func (c *Core) GetUserRoles() (Roles, error) {
	role, err := c.Roles.GetByCode(models.UserGroupRole)
	if err != nil {
		return Roles{}, err
	}
	return c.getGroupRoles(role.ID)
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

// getGroupRoles returns roles for group with specified ID.
func (c *Core) getGroupRoles(id int64) (Roles, error) {
	stack := []int64{id}
	roles := Roles{}
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
