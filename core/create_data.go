package core

import (
	"context"
	"fmt"

	"github.com/udovin/solve/models"
)

func CreateData(ctx context.Context, c *Core) error {
	fmt.Println("Creating default objects")
	roles := map[string]int64{}
	create := func(name string) error {
		role := models.Role{Name: name}
		if err := c.Roles.Create(ctx, &role); err != nil {
			return err

		}
		roles[role.Name] = role.ID
		return nil
	}
	join := func(child, parent string) error {
		edge := models.RoleEdge{
			RoleID:  roles[parent],
			ChildID: roles[child],
		}
		return c.RoleEdges.Create(ctx, &edge)
	}
	allRoles := models.GetBuiltInRoles()
	allGroups := []string{
		"guest_group",
		"user_group",
		"admin_group",
	}
	for _, role := range allRoles {
		if err := create(role); err != nil {
			return err
		}
	}
	for _, role := range allGroups {
		if err := create(role); err != nil {
			return err
		}
	}
	for _, role := range []string{
		models.LoginRole,
		models.RegisterRole,
		models.StatusRole,
		models.ObserveUserRole,
		models.ObserveProblemsRole,
		models.ObserveContestsRole,
		models.ObserveSolutionsRole,
	} {
		if err := join(role, "guest_group"); err != nil {
			return err
		}
	}
	for _, role := range []string{
		models.LoginRole,
		models.LogoutRole,
		models.StatusRole,
		models.ObserveUserRole,
		models.ObserveProblemsRole,
		models.ObserveContestsRole,
		models.ObserveSolutionsRole,
	} {
		if err := join(role, "user_group"); err != nil {
			return err
		}
	}
	for _, role := range allRoles {
		if err := join(role, "admin_group"); err != nil {
			return err
		}
	}
	return nil
}
