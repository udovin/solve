package config

import (
	"encoding/json"
	"fmt"
)

type StorageDriver string

const (
	LocalStorageDriver StorageDriver = "local"
	S3StorageDriver    StorageDriver = "s3"
)

type StorageOptions interface {
	Driver() StorageDriver
}

type LocalStorageOptions struct {
	FilesDir string `json:"files_dir"`
}

func (o LocalStorageOptions) Driver() StorageDriver {
	return LocalStorageDriver
}

type S3StorageOptions struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	PathPrefix      string `json:"path_prefix,omitempty"`
	UsePathStyle    bool   `json:"use_path_style,omitempty"`
}

func (o S3StorageOptions) Driver() StorageDriver {
	return S3StorageDriver
}

// Storage contains storage config.
type Storage struct {
	Options StorageOptions `json:"options"`
}

func (c Storage) MarshalJSON() ([]byte, error) {
	cfg := struct {
		Driver  StorageDriver  `json:"driver"`
		Options StorageOptions `json:"options"`
	}{
		Driver:  c.Options.Driver(),
		Options: c.Options,
	}
	return json.Marshal(cfg)
}

func (c *Storage) UnmarshalJSON(bytes []byte) error {
	var cfg struct {
		Driver  StorageDriver   `json:"driver"`
		Options json.RawMessage `json:"options"`
	}
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return err
	}
	switch cfg.Driver {
	case LocalStorageDriver:
		var options LocalStorageOptions
		if err := json.Unmarshal(cfg.Options, &options); err != nil {
			return err
		}
		c.Options = options
	case S3StorageDriver:
		var options S3StorageOptions
		if err := json.Unmarshal(cfg.Options, &options); err != nil {
			return err
		}
		c.Options = options
	default:
		return fmt.Errorf("driver %q is not supported", cfg.Driver)
	}
	return nil
}
