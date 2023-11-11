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
	createForm := CreateSettingForm{}
	createForm.Key = getPtr("test_key")
	createForm.Value = getPtr("test_value")
	setting, err := e.Client.CreateSetting(context.Background(), createForm)
	if err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(setting)
	}
	updateForm := UpdateSettingForm{}
	updateForm.Key = getPtr("test_key_2")
	updateForm.Value = getPtr("test_value_2")
	updated, err := e.Client.UpdateSetting(context.Background(), setting.ID, updateForm)
	if err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(updated)
	}
}
