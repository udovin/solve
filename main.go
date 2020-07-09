package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
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

// getConfig reads config with filename from '--config' flag.
func getConfig(cmd *cobra.Command) (config.Config, error) {
	filename, err := cmd.Flags().GetString("config")
	if err != nil {
		return config.Config{}, err
	}
	return config.LoadFromFile(filename)
}

// serverMain starts API server.
//
// Simply speaking this function does following things:
//  1. Setup Core instance (with all managers).
//  2. Setup Echo server instance.
//  3. Register API View to Echo server.
func serverMain(cmd *cobra.Command, _ []string) {
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	c, err := core.NewCore(cfg)
	if err != nil {
		panic(err)
	}
	if err := c.SetupAllManagers(); err != nil {
		panic(err)
	}
	if err := c.Start(); err != nil {
		panic(err)
	}
	defer c.Stop()
	// Create new echo server instance.
	s := echo.New()
	s.Logger = c.Logger()
	s.HideBanner = true
	s.HidePort = true
	// Setup middleware.
	s.Pre(middleware.RemoveTrailingSlash())
	s.Use(middleware.Recover())
	s.Use(middleware.Gzip())
	s.Use(middleware.Logger())
	// Create API view.
	v := api.NewView(c)
	// Register API view.
	v.Register(s.Group("/api/v0"))
	// Register view for static.
	s.Any("/*", func(c echo.Context) error {
		p, err := url.PathUnescape(c.Param("*"))
		if err != nil {
			return err
		}
		name := filepath.Join(cfg.Server.Static, path.Clean("/"+p))
		if _, err := os.Stat(name); os.IsNotExist(err) {
			name = filepath.Join(cfg.Server.Static, "index.html")
		}
		return c.File(name)
	})
	// Start echo server.
	if err := s.Start(cfg.Server.Address()); err != nil {
		s.Logger.Fatal(err)
	}
}

// invokerMain starts Invoker.
//
// This function initializes Core instance with only necessary
// stores for running actions.
func invokerMain(cmd *cobra.Command, _ []string) {
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	c, err := core.NewCore(cfg)
	if err != nil {
		panic(err)
	}
	c.SetupInvokerManagers()
	if err := c.Start(); err != nil {
		panic(err)
	}
	defer c.Stop()
	s := invoker.New(c)
	s.Start()
	wait := make(chan os.Signal)
	signal.Notify(wait, os.Interrupt, syscall.SIGTERM)
	<-wait
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
	if err := c.SetupAllManagers(); err != nil {
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
	if err := c.SetupAllManagers(); err != nil {
		panic(err)
	}
	if err := migrations.Unapply(c); err != nil {
		panic(err)
	}
}

// main is a main entry point.
//
// Solve is divided into two main parts:
//  * API server - server that provides HTTP API. If you want to
//    understand how API server is working you can go to serverMain
//    function.
//  * Invoker - server that performs asynchronous runs. See
//    invokerMain function if you want to start with Invoker.
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
		Use:   "invoker",
		Run:   invokerMain,
		Short: "Starts invoker daemon",
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
