package config

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/labstack/gommon/log"
)

type LogLevel log.Lvl

func (l LogLevel) MarshalText() ([]byte, error) {
	switch log.Lvl(l) {
	case 0:
		return nil, nil
	case log.DEBUG:
		return []byte("debug"), nil
	case log.INFO:
		return []byte("info"), nil
	case log.WARN:
		return []byte("warning"), nil
	case log.ERROR:
		return []byte("error"), nil
	case log.OFF:
		return []byte("off"), nil
	default:
		return nil, fmt.Errorf("unknown level %d", l)
	}
}

func (l *LogLevel) UnmarshalText(text []byte) error {
	switch string(text) {
	case "debug":
		*l = LogLevel(log.DEBUG)
	case "info":
		*l = LogLevel(log.INFO)
	case "warning", "warn":
		*l = LogLevel(log.WARN)
	case "error":
		*l = LogLevel(log.ERROR)
	case "off":
		*l = LogLevel(log.OFF)
	default:
		return fmt.Errorf("unknown level: %q", text)
	}
	return nil
}

var (
	_ encoding.TextMarshaler   = LogLevel(0)
	_ encoding.TextUnmarshaler = (*LogLevel)(nil)
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
	//  * debug
	//  * info (default)
	//  * warn
	//  * error
	//  * off
	LogLevel LogLevel `json:"log_level,omitempty"`
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
	// FilesDir contains path to files directory.
	FilesDir string `json:"files_dir"`
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
		LogLevel: LogLevel(log.INFO),
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
