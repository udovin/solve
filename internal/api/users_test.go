package api

import (
	"context"
	"testing"
)

func TestUserSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	if user, err := e.Client.ObserveUser(user.Login); err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(user)
	}
	if roles, err := e.Socket.ObserveUserRoles(context.Background(), user.Login); err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(roles)
	}
	user.LoginClient()
	defer user.LogoutClient()
	if status, err := e.Client.Status(); err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(status)
	}
	if user, err := e.Client.ObserveUser(user.Login); err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(user)
	}
}
