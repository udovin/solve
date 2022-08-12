package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"

	"github.com/udovin/solve/api"
	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/invoker"

	// Register DB migrations.
	_ "github.com/udovin/solve/migrations"
)

var shutdown = make(chan os.Signal, 1)

// getConfig reads config with filename from '--config' flag.
func getConfig(cmd *cobra.Command) (config.Config, error) {
	filename, err := cmd.Flags().GetString("config")
	if err != nil {
		return config.Config{}, err
	}
	return config.LoadFromFile(filename)
}

func isServerError(err error) bool {
	return err != nil && err != http.ErrServerClosed
}

func newServer(logger *core.Logger) *echo.Echo {
	srv := echo.New()
	srv.Logger = logger
	srv.HideBanner, srv.HidePort = true, true
	srv.Pre(middleware.RemoveTrailingSlash())
	srv.Use(middleware.Recover(), middleware.Gzip())
	return srv
}

// serverMain starts Solve server.
//
// Simply speaking this function does following things:
//  1. Setup Core instance (with all managers).
//  2. Setup Echo server instance (HTTP + unix socket).
//  3. Register API View to Echo server.
//  4. Start Invoker server.
func serverMain(cmd *cobra.Command, _ []string) {
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	if cfg.Server == nil && cfg.Invoker == nil {
		panic("section 'server' or 'invoker' should be configured")
	}
	c, err := core.NewCore(cfg)
	if err != nil {
		panic(err)
	}
	c.SetupAllStores()
	if err := c.Start(); err != nil {
		panic(err)
	}
	defer c.Stop()
	v := api.NewView(c)
	var waiter sync.WaitGroup
	defer waiter.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	waiter.Add(1)
	go func() {
		defer waiter.Done()
		select {
		case <-ctx.Done():
		case <-shutdown:
			cancel()
		}
	}()
	if file := cfg.SocketFile; file != "" {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			panic(err)
		}
		srv := newServer(c.Logger())
		if srv.Listener, err = net.Listen("unix", file); err != nil {
			panic(err)
		}
		v.RegisterSocket(srv.Group("/socket"))
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			defer cancel()
			if err := srv.Start(""); isServerError(err) {
				c.Logger().Error(err)
			}
		}()
		defer func() {
			if err := srv.Shutdown(context.Background()); err != nil {
				c.Logger().Error(err)
			}
		}()
	}
	if cfg.Server != nil {
		srv := newServer(c.Logger())
		v.Register(srv.Group("/api"))
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			defer cancel()
			if err := srv.Start(cfg.Server.Address()); isServerError(err) {
				c.Logger().Error(err)
			}
		}()
		defer func() {
			if err := srv.Shutdown(context.Background()); err != nil {
				c.Logger().Error(err)
			}
		}()
	}
	if cfg.Invoker != nil {
		if err := invoker.New(c).Start(); err != nil {
			panic(err)
		}
	}
	<-ctx.Done()
}

// clientMain applies changes on server.
func clientMain(cmd *cobra.Command, _ []string) {
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	if cfg.SocketFile == "" {
		panic("Socket file is not configured")
	}
	dialer := func(_ context.Context, _, _ string) (net.Conn, error) {
		return net.Dial("unix", cfg.SocketFile)
	}
	client := http.Client{
		Transport: &http.Transport{DialContext: dialer},
	}
	_ = client
}

func migrateMain(cmd *cobra.Command, args []string) {
	createData, err := cmd.Flags().GetBool("create-data")
	if err != nil {
		panic(err)
	}
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	c, err := core.NewCore(cfg)
	if err != nil {
		panic(err)
	}
	c.SetupAllStores()
	var options []db.MigrateOption
	if len(args) > 0 {
		options = append(options, db.WithMigration(args[0]))
	}
	if err := db.ApplyMigrations(context.Background(), c.DB, options...); err != nil {
		panic(err)
	}
	if len(args) == 0 && createData {
		if err := core.CreateData(context.Background(), c); err != nil {
			panic(err)
		}
	}
}

func versionMain(cmd *cobra.Command, _ []string) {
	println("solve version:", config.Version)
}

// main is a main entry point.
//
// Solve is divided into two main parts:
//  * API server - server that provides HTTP API (http + socket).
//  * Invoker - server that performs asynchronous runs.
// This two parts was running from serverMain with respect of configuration.
// API server will be run if "server" section was specified.
// Invoker will be run if "invoker" section was specified.
//
// Also Solve has CLI interface like 'migrate'. This is a group of commands
// that work with database migrations.
func main() {
	rootCmd := cobra.Command{Use: os.Args[0]}
	rootCmd.PersistentFlags().String("config", "config.json", "")
	rootCmd.AddCommand(&cobra.Command{
		Use:   "server",
		Run:   serverMain,
		Short: "Starts API server",
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "client",
		Run:   clientMain,
		Short: "Commands for managing server",
	})
	migrateCmd := cobra.Command{
		Use:   "migrate",
		Run:   migrateMain,
		Short: "Applies migrations to database",
	}
	migrateCmd.Flags().Bool("create-data", false, "Create default objects")
	rootCmd.AddCommand(&migrateCmd)
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Run:   versionMain,
		Short: "Prints information about version",
	})
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
