package config

type SecurityConfig struct {
	PasswordSalt Secret `json:""`
}
