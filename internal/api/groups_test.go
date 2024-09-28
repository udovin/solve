package api

import (
	"context"
	"testing"
)

func TestGroupSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles("observe_groups", "create_group")
	user.LoginClient()
	defer user.LogoutClient()
	form := CreateGroupForm{
		Title: getPtr("Test group"),
	}
	if group, err := e.Client.CreateGroup(context.Background(), form); err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(group)
	}
}
