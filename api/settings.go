package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/models"
)

type Setting struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Settings struct {
	Settings []Setting `json:"settings"`
}

func makeSetting(o models.Setting) Setting {
	return Setting{
		ID:    o.ID,
		Key:   o.Key,
		Value: o.Value,
	}
}

// registerSettingHandlers registers handlers for settings management.
func (v *View) registerSettingHandlers(g *echo.Group) {
	g.GET(
		"/v0/settings", v.observeSettings,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(models.ObserveSettingsRole),
	)
	g.POST(
		"/v0/settings", v.createSetting,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.CreateSettingRole),
	)
	g.PATCH(
		"/v0/settings/:setting", v.updateSetting,
		v.extractAuth(v.sessionAuth), v.extractSetting,
		v.requirePermission(models.UpdateSettingRole),
	)
	g.DELETE(
		"/v0/settings/:setting", v.deleteSetting,
		v.extractAuth(v.sessionAuth), v.extractSetting,
		v.requirePermission(models.DeleteSettingRole),
	)
}

func (v *View) observeSettings(c echo.Context) error {
	if err := syncStore(c, v.core.Settings); err != nil {
		return err
	}
	var resp Settings
	settings, err := v.core.Settings.All()
	if err != nil {
		return err
	}
	for _, setting := range settings {
		resp.Settings = append(resp.Settings, makeSetting(setting))
	}
	sortFunc(resp.Settings, settingLess)
	return c.JSON(http.StatusOK, resp)
}

type updateSettingForm struct {
	Key   *string `json:"key"`
	Value *string `json:"value"`
}

func (f *updateSettingForm) Update(c echo.Context, o *models.Setting) error {
	if f.Key != nil {
		o.Key = *f.Key
	}
	if f.Value != nil {
		o.Value = *f.Value
	}
	return nil
}

type createSettingForm updateSettingForm

func (f *createSettingForm) Update(c echo.Context, o *models.Setting) error {
	if f.Key == nil {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Setting key cannot be empty."),
		}
	}
	return (*updateSettingForm)(f).Update(c, o)
}

func (v *View) createSetting(c echo.Context) error {
	var form createSettingForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var setting models.Setting
	if err := form.Update(c, &setting); err != nil {
		return err
	}
	if err := v.core.Settings.Create(getContext(c), &setting); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeSetting(setting))
}

func (v *View) updateSetting(c echo.Context) error {
	setting, ok := c.Get(settingKey).(models.Setting)
	if !ok {
		return fmt.Errorf("setting not extracted")
	}
	var form updateSettingForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := form.Update(c, &setting); err != nil {
		return err
	}
	if err := v.core.Settings.Update(getContext(c), setting); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeSetting(setting))
}

func (v *View) deleteSetting(c echo.Context) error {
	setting, ok := c.Get(settingKey).(models.Setting)
	if !ok {
		return fmt.Errorf("setting not extracted")
	}
	if err := v.core.Settings.Delete(getContext(c), setting.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, makeSetting(setting))
}

func getSettingByParam(
	c echo.Context,
	settings *models.SettingStore,
	key string,
) (models.Setting, error) {
	id, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return settings.GetByKey(key)
	}
	return settings.Get(getContext(c), id)
}

func (v *View) extractSetting(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		key := c.Param("setting")
		if err := syncStore(c, v.core.Settings); err != nil {
			return err
		}
		setting, err := getSettingByParam(c, v.core.Settings, key)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Setting not found."),
				}
			}
			return err
		}
		c.Set(settingKey, setting)
		return next(c)
	}
}

func settingLess(l, r Setting) bool {
	return l.Key < r.Key
}
