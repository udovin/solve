package core

import (
	"github.com/udovin/solve/models"
)

// RoleSet contains role set.
type RoleSet map[int64]struct{}

// HasRole return that role set has specified role.
func (s RoleSet) HasRole(id int64) bool {
	_, ok := s[id]
	return ok
}

// Clone creates clone of role set.
func (s RoleSet) Clone() RoleSet {
	var clone RoleSet
	for key := range s {
		clone[key] = struct{}{}
	}
	return clone
}

// AddRole adds role to role set.
func (c *Core) AddRole(roles RoleSet, code string) error {
	role, err := c.Roles.GetByCode(code)
	if err != nil {
		return err
	}
	roles[role.ID] = struct{}{}
	return nil
}

// HasRole checks that role set has specified role.
func (c *Core) HasRole(roles RoleSet, code string) (bool, error) {
	role, err := c.Roles.GetByCode(code)
	if err != nil {
		return false, err
	}
	return roles.HasRole(role.ID), nil
}

// GetGuestRoles returns roles for guest account.
func (c *Core) GetGuestRoles() (RoleSet, error) {
	role, err := c.Roles.GetByCode(models.GuestGroupRole)
	if err != nil {
		return nil, err
	}
	return c.getRecursiveRoles(role.ID)
}

// GetAccountRoles returns roles for account.
func (c *Core) GetAccountRoles(id int64) (RoleSet, error) {
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
func (c *Core) getRecursiveRoles(ids ...int64) (RoleSet, error) {
	stack, roles := ids, RoleSet{}
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
