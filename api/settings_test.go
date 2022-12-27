package api

import (
	"context"
	"testing"
)

func TestObserveSettings(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles("observe_settings", "create_setting", "update_setting", "delete_setting")
	user.LoginClient()
	defer user.LogoutClient()
	settings, err := e.Client.ObserveSettings(context.Background())
	if err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(settings)
	}
}
