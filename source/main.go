package main

import (
	"errors"
	"os"
	"path/filepath"

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
	server, err := core.NewServer(&cfg.Server)
	if err != nil {
		panic(err)
	}
	api.Register(app, server)
	if err := server.Listen(); err != nil {
		panic(err)
	}
}
