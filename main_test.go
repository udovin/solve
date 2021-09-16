package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/migrations"
)

var (
	testConfigFile *os.File
	testConfig     = config.Config{
		DB: config.DB{
			Driver:  config.SQLiteDriver,
			Options: config.SQLiteOptions{Path: ":memory:?cache=shared"},
		},
		SocketFile: fmt.Sprintf("/tmp/test-solve-%d.sock", rand.Int63()),
		Server:     &config.Server{},
		Invoker:    &config.Invoker{},
		Security: config.Security{
			PasswordSalt: config.Secret{
				Type: config.DataSecret,
				Data: "qwerty123",
			},
		},
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func testSetup(tb testing.TB) {
	var err error
	func() {
		testConfigFile, err = ioutil.TempFile("", "test-")
		if err != nil {
			tb.Fatal("Error:", err)
		}
		defer testConfigFile.Close()
		err := json.NewEncoder(testConfigFile).Encode(testConfig)
		if err != nil {
			tb.Fatal("Error:", err)
		}
	}()
	c, err := core.NewCore(testConfig)
	if err != nil {
		tb.Fatal("Error:", err)
	}
	if err := c.SetupAllStores(); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := migrations.Apply(c); err != nil {
		tb.Fatal("Error:", err)
	}
}

func testTeardown(tb testing.TB) {
	os.RemoveAll(testConfigFile.Name())
	c, err := core.NewCore(testConfig)
	if err != nil {
		tb.Fatal("Error:", err)
	}
	if err := c.SetupAllStores(); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := migrations.Unapply(c); err != nil {
		tb.Fatal("Error:", err)
	}
}

func TestServerMain(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	cmd := cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Set("config", testConfigFile.Name())
	go func() {
		shutdown <- os.Interrupt
	}()
	serverMain(&cmd, nil)
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

func TestDBApplyMain(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	cmd := cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Set("config", testConfigFile.Name())
	go func() {
		shutdown <- os.Interrupt
	}()
	dbApplyMain(&cmd, nil)
}

func TestDBUnapplyMain(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	cmd := cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Set("config", testConfigFile.Name())
	go func() {
		shutdown <- os.Interrupt
	}()
	dbUnapplyMain(&cmd, nil)
}

func TestGetConfigUnknown(t *testing.T) {
	cmd := cobra.Command{}
	if _, err := getConfig(&cmd); err == nil {
		t.Fatal("Expected error")
	}
}

func TestCommand(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic")
		}
	}()
	args := os.Args
	os.Args = []string{"solve", "--config", "not-found", "server"}
	main()
	os.Args = args
}
