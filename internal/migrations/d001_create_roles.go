package migrations

import (
	"context"
	"database/sql"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
)

func init() {
	Data.AddMigration("001_create_roles", d001{})
}

type d001 struct{}

type FindQuery = db.FindQuery

func (m d001) Apply(ctx context.Context, db *gosql.DB) error {
	roleStore := models.NewRoleStore(db, "solve_role", "solve_role_event")
	roleEdgeStore := models.NewRoleEdgeStore(db, "solve_role_edge", "solve_role_edge_event")
	roles := map[string]int64{}
	create := func(name string) error {
		if role, err := roleStore.FindOne(ctx, FindQuery{
			Where: gosql.Column("name").Equal(name),
		}); err != nil {
			if err != sql.ErrNoRows {
				return err
			}
		} else {
			roles[role.Name] = role.ID
			return nil
		}
		role := models.Role{Name: name}
		if err := roleStore.Create(ctx, &role); err != nil {
			return err
		}
		roles[role.Name] = role.ID
		return nil
	}
	join := func(child, parent string) error {
		if _, err := roleEdgeStore.FindOne(ctx, FindQuery{
			Where: gosql.Column("role_id").Equal(roles[parent]).
				And(gosql.Column("child_id").Equal(roles[child])),
		}); err != nil {
			if err != sql.ErrNoRows {
				return err
			}
		} else {
			return nil
		}
		edge := models.RoleEdge{
			RoleID:  roles[parent],
			ChildID: roles[child],
		}
		return roleEdgeStore.Create(ctx, &edge)
	}
	allRoles := perms.GetBuiltInRoles()
	allGroups := []string{
		"guest_group",
		"pending_user_group",
		"active_user_group",
		"blocked_user_group",
		"scope_user_group",
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
	baseRoles := []string{
		perms.LoginRole,
		perms.StatusRole,
		perms.ObserveUserRole,
		perms.ObserveContestsRole,
		perms.ObserveCompilersRole,
		perms.ObservePostsRole,
		perms.ConsumeTokenRole,
		perms.ResetPasswordRole,
	}
	getGroupRoles := func(roles ...string) []string {
		return append(roles, baseRoles...)
	}
	for _, role := range getGroupRoles(perms.RegisterRole) {
		if err := join(role, "guest_group"); err != nil {
			return err
		}
	}
	for _, role := range getGroupRoles(perms.LogoutRole) {
		if err := join(role, "pending_user_group"); err != nil {
			return err
		}
	}
	for _, role := range getGroupRoles(perms.LogoutRole, perms.RegisterContestsRole) {
		if err := join(role, "active_user_group"); err != nil {
			return err
		}
	}
	for _, role := range getGroupRoles(perms.LogoutRole) {
		if err := join(role, "blocked_user_group"); err != nil {
			return err
		}
	}
	for _, role := range getGroupRoles(perms.LogoutRole) {
		if err := join(role, "scope_user_group"); err != nil {
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
