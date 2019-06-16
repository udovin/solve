package main

import (
	"errors"
	"os"
	"path/filepath"

	"./config"
	"./http"
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
	server, err := http.NewServer(&cfg)
	if err != nil {
		panic(err)
	}
	if err := server.Listen(); err != nil {
		panic(err)
	}
}
