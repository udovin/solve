package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/spf13/cobra"

	"github.com/udovin/solve/api"
	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/invoker"

	_ "github.com/opencontainers/runc/libcontainer/nsenter"
)

func init() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runtime.GOMAXPROCS(1)
		runtime.LockOSThread()
		factory, _ := libcontainer.New("")
		if err := factory.StartInitialization(); err != nil {
			panic(err)
		}
	}
}

func getConfig(cmd *cobra.Command) (config.Config, error) {
	path, err := cmd.Flags().GetString("config")
	if err != nil {
		return config.Config{}, err
	}
	return config.LoadFromFile(path)
}

func getAddress(cfg config.Server) string {
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
}

func serverMain(cmd *cobra.Command, args []string) {
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	app, err := core.NewApp(&cfg)
	if err != nil {
		panic(err)
	}
	if err := app.Start(); err != nil {
		panic(err)
	}
	defer app.Stop()
	server := echo.New()
	server.Use(middleware.Recover())
	server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339}\t${latency_human}\t${status}\t${method}\t${uri}\n",
	}))
	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(middleware.Gzip())
	api.Register(app, server)
	server.Logger.Fatal(server.Start(getAddress(cfg.Server)))
}

func invokerMain(cmd *cobra.Command, args []string) {
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	app, err := core.NewApp(&cfg)
	if err != nil {
		panic(err)
	}
	if err := app.Start(); err != nil {
		panic(err)
	}
	defer app.Stop()
	server := invoker.New(app)
	server.Start()
	defer server.Stop()
	wait := make(chan os.Signal)
	signal.Notify(wait, os.Interrupt, syscall.SIGTERM)
	<-wait
}

func main() {
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
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
