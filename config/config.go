package config

import (
	"encoding/json"
	"io/ioutil"
)

// Config stores configuration for Solve API and Invoker
type Config struct {
	Server   Server   `json:""`
	DB       DB       `json:""`
	Security Security `json:""`
}

// LoadFromFile loads configuration from json file
func LoadFromFile(file string) (cfg Config, err error) {
	bytes, err := ioutil.ReadFile(file)
	if err == nil {
		err = json.Unmarshal(bytes, &cfg)
	}
	return
}
