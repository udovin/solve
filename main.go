package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
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
	if err := app.SetupAllManagers(); err != nil {
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
	api.Register(app, server.Group("/api/v0"))
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
	app.SetupInvokerManagers()
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

func upgradeDbMain(cmd *cobra.Command, args []string) {
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	db, err := cfg.DB.Create()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = db.Close()
	}()
	files, err := ioutil.ReadDir("migrations")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		fmt.Println(file.Name())
	}
}

func main() {
	// Setup good logs
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
