package migrations

import (
	"context"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/models"
)

func init() {
	Data.AddMigration("001_create_roles", d001{})
}

type d001 struct{}

func (m d001) Apply(ctx context.Context, db *gosql.DB) error {
	roleStore := models.NewRoleStore(db, "solve_role", "solve_role_event")
	roleEdgeStore := models.NewRoleEdgeStore(db, "solve_role_edge", "solve_role_edge_event")
	roles := map[string]int64{}
	create := func(name string) error {
		role := models.Role{Name: name}
		if err := roleStore.Create(ctx, &role); err != nil {
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
		return roleEdgeStore.Create(ctx, &edge)
	}
	allRoles := models.GetBuiltInRoles()
	allGroups := []string{
		"guest_group",
		"pending_user_group",
		"active_user_group",
		"blocked_user_group",
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
		models.ObserveContestsRole,
		models.ObserveCompilersRole,
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
		models.ObserveContestsRole,
		models.ObserveCompilersRole,
	} {
		if err := join(role, "pending_user_group"); err != nil {
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
		models.ObserveCompilersRole,
		models.RegisterContestsRole,
	} {
		if err := join(role, "active_user_group"); err != nil {
			return err
		}
	}
	for _, role := range []string{
		models.LoginRole,
		models.LogoutRole,
		models.StatusRole,
		models.ObserveUserRole,
		models.ObserveContestsRole,
		models.ObserveCompilersRole,
	} {
		if err := join(role, "blocked_user_group"); err != nil {
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

func (m d001) Unapply(ctx context.Context, db *gosql.DB) error {
	return nil
}
