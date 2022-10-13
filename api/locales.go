package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (v *View) registerLocaleHandlers(g *echo.Group) {
	g.GET(
		"/v0/locale", v.currentLocale,
		v.extractAuth(v.sessionAuth, v.guestAuth),
	)
}

type Localization struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type Locale struct {
	Name          string         `json:"name,omitempty"`
	Localizations []Localization `json:"localizations,omitempty"`
}

func (v *View) currentLocale(c echo.Context) error {
	locale := getLocale(c)
	localizations, err := locale.GetLocalizations()
	if err != nil {
		return err
	}
	resp := Locale{
		Name:          locale.Name(),
		Localizations: localizations,
	}
	sortFunc(resp.Localizations, localizationLess)
	return c.JSON(http.StatusOK, resp)
}

func localizationLess(lhs, rhs Localization) bool {
	return lhs.Key < rhs.Key
}
