package config

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap/zapcore"
	"io/ioutil"

	"go.uber.org/zap"
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
	// Logger contains logger configuration.
	Logger Logger `json:"logger"`
}

// Server contains server config.
type Server struct {
	// Host contains server host.
	Host string `json:"host"`
	// Port contains server port.
	Port int `json:"port"`
	// SocketFile contains path to socket.
	SocketFile string `json:"socket_file"`
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

type Logger struct {
	Level zapcore.Level `json:"level"`
}

// LoadFromFile loads configuration from json file.
func LoadFromFile(file string) (Config, error) {
	cfg := Config{
		Server: Server{
			SocketFile: "/tmp/solve-server.sock",
		},
		Logger: Logger{
			// By default we should use INFO level.
			Level: zap.InfoLevel,
		},
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
