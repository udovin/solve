package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"./api"
	"./config"
	"./core"
)

const EtcDir = "/etc/solve"

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func getConfig() (config.Config, error) {
	path, ok := os.LookupEnv("SOLVE_CONFIG_FILE")
	if ok {
		return config.LoadFromFile(path)
	}
	path = "config.json"
	if fileExists(path) {
		return config.LoadFromFile(path)
	}
	path = filepath.Join(EtcDir, path)
	if fileExists(path) {
		return config.LoadFromFile(path)
	}
	return config.Config{}, errors.New("unable to find config file")
}

func getAddress(cfg config.ServerConfig) string {
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		panic(err)
	}
	app, err := core.NewApp(&cfg)
	if err != nil {
		panic(err)
	}
	app.Start()
	defer app.Stop()
	server := echo.New()
	server.Use(middleware.Recover())
	server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339}\t${latency_human}\t${status}\t${method}\t${uri}\n",
	}))
	server.Use(middleware.Gzip())
	server.Static("/static", "static")
	api.Register(app, server)
	server.Pre(middleware.RemoveTrailingSlash())
	server.Logger.Fatal(server.Start(fmt.Sprintf(
		"%s:%d", cfg.Server.Host, cfg.Server.Port,
	)))
}
