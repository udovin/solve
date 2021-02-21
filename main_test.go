package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestServerMain(t *testing.T) {
	cmd := cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Set("config", "not-found")
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected panic")
		}
	}()
	serverMain(&cmd, nil)
}

func TestInvokerMain(t *testing.T) {
	cmd := cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Set("config", "not-found")
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected panic")
		}
	}()
	invokerMain(&cmd, nil)
}

func TestClientMain(t *testing.T) {
	cmd := cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Set("config", "not-found")
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected panic")
		}
	}()
	clientMain(&cmd, nil)
}
