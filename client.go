package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/udovin/solve/api"
)

var ClientCmd = cobra.Command{
	Use: "client",
}

func init() {
	// Users.
	createUserCmd := cobra.Command{
		Use:  "create-user",
		RunE: wrapClientMain(createUserMain),
	}
	createUserCmd.Flags().String("login", "", "")
	createUserCmd.Flags().String("password", "", "")
	createUserCmd.Flags().String("email", "", "")
	createUserCmd.Flags().StringArray("add-role", nil, "")
	createUserCmd.MarkFlagRequired("login")
	createUserCmd.MarkFlagRequired("password")
	createUserCmd.MarkFlagRequired("email")
	ClientCmd.AddCommand(&createUserCmd)
	// Roles.
	addRoleCmd := cobra.Command{
		Use: "add-role",
	}
	addRoleCmd.Flags().String("name", "", "")
	ClientCmd.AddCommand(&addRoleCmd)
	//
	deleteRoleCmd := cobra.Command{
		Use: "delete-role",
	}
	deleteRoleCmd.Flags().String("name", "", "")
	ClientCmd.AddCommand(&deleteRoleCmd)
	//
	addChildRoleCmd := cobra.Command{
		Use: "add-child-role",
	}
	addChildRoleCmd.Flags().String("role", "", "")
	addChildRoleCmd.Flags().StringArray("child", nil, "")
	ClientCmd.AddCommand(&addChildRoleCmd)
	//
	deleteChildRoleCmd := cobra.Command{
		Use: "delete-child-role",
	}
	deleteChildRoleCmd.Flags().String("role", "", "")
	deleteChildRoleCmd.Flags().StringArray("child", nil, "")
	ClientCmd.AddCommand(&deleteChildRoleCmd)
	// User roles.
	addUserRoleCmd := cobra.Command{
		Use: "add-user-role",
	}
	addUserRoleCmd.Flags().String("user", "", "")
	addUserRoleCmd.Flags().StringArray("role", nil, "")
	ClientCmd.AddCommand(&addUserRoleCmd)
	//
	deleteUserRoleCmd := cobra.Command{
		Use: "delete-user-role",
	}
	deleteUserRoleCmd.Flags().String("user", "", "")
	deleteUserRoleCmd.Flags().StringArray("role", nil, "")
	ClientCmd.AddCommand(&deleteUserRoleCmd)
}

func createUserMain(ctx *clientContext) error {
	login := must(ctx.Cmd.Flags().GetString("login"))
	email := must(ctx.Cmd.Flags().GetString("email"))
	password := must(ctx.Cmd.Flags().GetString("password"))
	addRoles := must(ctx.Cmd.Flags().GetStringArray("add-role"))
	_, err := ctx.Client.Register(context.Background(), api.RegisterUserForm{
		Login:    login,
		Email:    email,
		Password: password,
	})
	if err != nil {
		return fmt.Errorf("unable to register user: %w", err)
	}
	time.Sleep(2 * time.Second)
	for _, role := range addRoles {
		if _, err := ctx.Client.CreateUserRole(context.Background(), login, role); err != nil {
			return fmt.Errorf("unable to add role %q: %w", role, err)
		}
	}
	return nil
}

type clientContext struct {
	Cmd    *cobra.Command
	Args   []string
	Client *api.Client
}

func wrapClientMain(fn func(*clientContext) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := clientContext{
			Cmd:  cmd,
			Args: args,
		}
		config, err := getConfig(cmd)
		if err != nil {
			return err
		}
		transport := http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", config.SocketFile)
			},
		}
		ctx.Client = api.NewClient("http://server/socket", api.WithTransport(&transport))
		return fn(&ctx)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
