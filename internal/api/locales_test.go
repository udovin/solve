package api

import (
	"context"
	"testing"
)

func TestCurrentLocale(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	if locale, err := e.Client.Locale(context.Background()); err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(locale)
	}
	e.Client.Headers = map[string]string{"Accept-Language": "en"}
	if locale, err := e.Client.Locale(context.Background()); err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(locale)
	}
	e.Client.Headers = map[string]string{"Accept-Language": "ru"}
	if locale, err := e.Client.Locale(context.Background()); err != nil {
		t.Fatal("Error:", err)
	} else {
		e.Check(locale)
	}
}
