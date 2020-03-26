package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Config stores configuration for Solve API and Invoker.
type Config struct {
	// DB contains database connection config.
	DB DB `json:""`
	// Server contains API server config.
	Server Server `json:""`
	// Invoker contains invoker config.
	Invoker Invoker `json:""`
	// Security contains security config.
	Security Security `json:""`
}

// Server contains server config.
type Server struct {
	Host   string `json:""`
	Port   int    `json:""`
	Static string `json:""`
}

// Address returns string representation of server address.
func (s Server) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Security contains security config.
type Security struct {
	PasswordSalt Secret `json:""`
}

// Invoker contains invoker config.
type Invoker struct {
	ProblemsDir string `json:""`
	Threads     int    `json:""`
}

// LoadFromFile loads configuration from json file.
func LoadFromFile(file string) (cfg Config, err error) {
	bytes, err := ioutil.ReadFile(file)
	if err == nil {
		err = json.Unmarshal(bytes, &cfg)
	}
	return
}
