package api

import (
	"context"
	"testing"

	"github.com/udovin/solve/internal/perms"
)

func TestAccountsSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles(perms.ObserveAccountsRole)
	user.LoginClient()
	{
		accounts, err := e.Client.ObserveAccounts(context.Background())
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(accounts)
	}
}
