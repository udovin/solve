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
)

func getConfig(cmd *cobra.Command) (config.Config, error) {
	cfgPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return config.Config{}, err
	}
	return config.LoadFromFile(cfgPath)
}

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
			return c.File(filepath.Join(cfg.Server.Static, "index.html"))
		} else {
			return c.File(name)
		}
	})
	// Start echo server.
	if err := s.Start(cfg.Server.Address()); err != nil {
		s.Logger.Fatal(err)
	}
}

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
	defer s.Stop()
	wait := make(chan os.Signal)
	signal.Notify(wait, os.Interrupt, syscall.SIGTERM)
	<-wait
}

func main() {
	// Setup good logs.
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	rootCmd := cobra.Command{}
	rootCmd.PersistentFlags().String("config", "config.json", "")
	rootCmd.AddCommand(&cobra.Command{
		Use: "server",
		Run: serverMain,
	})
	rootCmd.AddCommand(&cobra.Command{
		Use: "invoker",
		Run: invokerMain,
	})
	dbCmd := cobra.Command{
		Use: "db",
	}
	dbCmd.AddCommand(&cobra.Command{
		Use: "upgrade",
		Run: upgradeDbMain,
	})
	rootCmd.AddCommand(&dbCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
