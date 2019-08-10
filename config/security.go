package config

type Security struct {
	PasswordSalt Secret `json:""`
}
