package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/labstack/gommon/log"
)

// Config stores configuration for Solve API and Invoker.
type Config struct {
	// DB contains database connection config.
	DB DB `json:"db"`
	// Server contains API server config.
	Server Server `json:"server"`
	// Invoker contains invoker config.
	Invoker Invoker `json:"invoker"`
	// Security contains security config.
	Security Security `json:"security"`
	// LogLevel contains level of logging.
	//
	// You can use following values:
	//  * 1 - DEBUG
	//  * 2 - INFO (default)
	//  * 3 - WARN
	//  * 4 - ERROR
	//  * 5 - OFF
	LogLevel log.Lvl `json:"log_level"`
}

// Server contains server config.
type Server struct {
	// Host contains server host.
	Host string `json:"host"`
	// Port contains server port.
	Port int `json:"port"`
}

// Address returns string representation of server address.
func (s Server) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Security contains security config.
type Security struct {
	// PasswordSalt contains salt for password hashing.
	PasswordSalt Secret `json:"password_salt"`
}

// Invoker contains invoker config.
type Invoker struct {
	ProblemsDir string `json:"problems_dir"`
	Threads     int    `json:"threads"`
}

// LoadFromFile loads configuration from json file.
func LoadFromFile(file string) (Config, error) {
	cfg := Config{
		// By default we should use INFO level.
		LogLevel: log.INFO,
	}
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return Config{}, err
	}
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
