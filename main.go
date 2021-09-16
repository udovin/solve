package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/spf13/cobra"

	"github.com/udovin/solve/api"
	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/invoker"
	"github.com/udovin/solve/migrations"
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
	if err := c.SetupAllStores(); err != nil {
		panic(err)
	}
	if err := c.Start(); err != nil {
		panic(err)
	}
	defer c.Stop()
	v := api.NewView(c)
	var waiter sync.WaitGroup
	defer waiter.Wait()
	ctx, cancel := context.WithCancel(context.Background())
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
		sock := echo.New()
		if sock.Listener, err = net.Listen("unix", file); err != nil {
			panic(err)
		}
		sock.Logger = c.Logger()
		sock.HideBanner, sock.HidePort = true, true
		sock.Pre(middleware.RemoveTrailingSlash())
		sock.Use(middleware.Recover(), middleware.Gzip(), middleware.Logger())
		v.RegisterSocket(sock.Group("/api/v0"))
		waiter.Add(1)
		go func() {
			defer waiter.Done()
			defer cancel()
			if err := sock.Start(""); isServerError(err) {
				c.Logger().Error(err)
			}
		}()
		defer func() {
			if err := sock.Shutdown(context.Background()); err != nil {
				c.Logger().Error(err)
			}
		}()
	}
	if cfg.Server != nil {
		srv := echo.New()
		srv.Logger = c.Logger()
		srv.HideBanner, srv.HidePort = true, true
		srv.Pre(middleware.RemoveTrailingSlash())
		srv.Use(middleware.Recover(), middleware.Gzip(), middleware.Logger())
		v.Register(srv.Group("/api/v0"))
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
		invoker.New(c).Start()
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

func dbApplyMain(cmd *cobra.Command, _ []string) {
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	c, err := core.NewCore(cfg)
	if err != nil {
		panic(err)
	}
	if err := c.SetupAllStores(); err != nil {
		panic(err)
	}
	if err := migrations.Apply(c); err != nil {
		panic(err)
	}
}

func dbUnapplyMain(cmd *cobra.Command, _ []string) {
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	c, err := core.NewCore(cfg)
	if err != nil {
		panic(err)
	}
	if err := c.SetupAllStores(); err != nil {
		panic(err)
	}
	if err := migrations.Unapply(c); err != nil {
		panic(err)
	}
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
// Also Solve has CLI interface like 'db'. This is a group of commands
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
	dbCmd := cobra.Command{
		Use:   "db",
		Short: "Commands for managing database",
	}
	dbCmd.AddCommand(&cobra.Command{
		Use:   "apply",
		Run:   dbApplyMain,
		Short: "Applies all new migrations to database",
	})
	dbCmd.AddCommand(&cobra.Command{
		Use:   "unapply",
		Run:   dbUnapplyMain,
		Short: "Rolls back all applied migrations",
	})
	rootCmd.AddCommand(&dbCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
