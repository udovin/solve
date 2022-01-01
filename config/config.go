package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/labstack/gommon/log"
)

// Config stores configuration for Solve API and Invoker.
type Config struct {
	// DB contains database connection config.
	DB DB `json:"db"`
	// SocketFile contains path to socket.
	SocketFile string `json:"socket_file"`
	// Server contains API server config.
	Server *Server `json:"server"`
	// Invoker contains invoker config.
	Invoker *Invoker `json:"invoker"`
	// Storage contains configuration for storage.
	Storage *Storage `json:"storage"`
	// Security contains security config.
	Security *Security `json:"security"`
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
	PasswordSalt string `json:"password_salt"`
}

// Invoker contains invoker config.
type Invoker struct {
	// Threads contains amount of parallel workers.
	Threads int `json:"threads"`
}

// Storage contains storage config.
type Storage struct {
	// ProblemsDir contains path to problems directory.
	ProblemsDir string `json:"problems_dir"`
	// SolutionsDir contains path to solutions directory.
	SolutionsDir string `json:"solutions_dir"`
	// CompilersDir contains path to compilers directory.
	CompilersDir string `json:"compilers_dir"`
}

var configFuncs = template.FuncMap{
	"json": func(value interface{}) (string, error) {
		data, err := json.Marshal(value)
		return string(data), err
	},
	"file": func(name string) (string, error) {
		bytes, err := ioutil.ReadFile(name)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(bytes), "\r\n"), nil
	},
	"env": os.Getenv,
}

// LoadFromFile loads configuration from json file.
func LoadFromFile(file string) (Config, error) {
	cfg := Config{
		SocketFile: "/tmp/solve-server.sock",
		// By default we should use INFO level.
		LogLevel: log.INFO,
	}
	tmpl, err := template.New(filepath.Base(file)).
		Funcs(configFuncs).ParseFiles(file)
	if err != nil {
		return Config{}, err
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, nil); err != nil {
		return Config{}, err
	}
	if err := json.NewDecoder(&buffer).Decode(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
