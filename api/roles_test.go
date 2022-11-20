package api

import (
	"context"
	"fmt"
	"testing"
)

func TestObserveRoles(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	roles, err := e.Socket.ObserveRoles(context.Background())
	if err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(roles)
	}
}

func TestCreateDeleteRole(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	created, err := e.Socket.CreateRole(context.Background(), "test_role")
	if err != nil {
		t.Error("Error:", err)
	}
	e.Check(created)
	deleted, err := e.Socket.DeleteRole(context.Background(), created.ID)
	if err != nil {
		t.Error("Error:", err)
	}
	if created != deleted {
		t.Fatal("Invalid deleted role:", deleted)
	}
}

func TestRoleSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	for i := 1; i < 5; i++ {
		role, err := e.Socket.CreateRole(context.Background(), fmt.Sprintf("role%d", i))
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(role)
	}
	user := NewTestUser(e)
	user.LoginClient()
	user.AddRoles("admin_group")
	for i := 2; i < 5; i++ {
		roles, err := e.Client.CreateRoleRole("role1", fmt.Sprintf("role%d", i))
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(roles)
	}
	for i := 2; i < 5; i++ {
		roles, err := e.Client.DeleteRoleRole("role1", fmt.Sprintf("role%d", i))
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(roles)
	}
	{
		if _, err := e.Client.DeleteRoleRole("role1", "role2"); err == nil {
			t.Fatal("Expected error")
		} else {
			e.Check(err)
		}
		if _, err := e.Client.DeleteRoleRole("role1", "role100"); err == nil {
			t.Fatal("Expected error")
		} else {
			e.Check(err)
		}
	}
	for i := 1; i < 5; i++ {
		roles, err := e.Client.CreateUserRole(context.Background(), user.Login, fmt.Sprintf("role%d", i))
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(roles)
	}
	for i := 1; i < 5; i++ {
		roles, err := e.Client.DeleteUserRole(context.Background(), user.Login, fmt.Sprintf("role%d", i))
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(roles)
	}
	{
		if _, err := e.Client.DeleteUserRole(context.Background(), user.Login, "role2"); err == nil {
			t.Fatal("Expected error")
		} else {
			e.Check(err)
		}
		if _, err := e.Client.DeleteUserRole(context.Background(), user.Login, "role100"); err == nil {
			t.Fatal("Expected error")
		} else {
			e.Check(err)
		}
		if _, err := e.Client.DeleteUserRole(context.Background(), "user100", "role2"); err == nil {
			t.Fatal("Expected error")
		} else {
			e.Check(err)
		}
	}
}
