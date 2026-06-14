package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

const (
	defaultConfigFileName = "revinder_bridge.json"
	defaultDatabaseName   = "revinder_bridge.sqlite"
	defaultBindAddress    = "*"
	defaultPort           = 8080
)

type Config struct {
	BindAddress  string `json:"bind_address"`
	Port         int    `json:"port"`
	DatabasePath string `json:"database_path"`
}

func Load() (Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return Config{}, err
	}

	binaryDir := filepath.Dir(exePath)
	cfg := Config{
		BindAddress:  defaultBindAddress,
		Port:         defaultPort,
		DatabasePath: filepath.Join(binaryDir, defaultDatabaseName),
	}

	configPath := filepath.Join(binaryDir, defaultConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fileCfg := cfg
			fileCfg.DatabasePath = defaultDatabaseName
			if err := writeDefaultConfig(configPath, fileCfg); err != nil {
				return Config{}, err
			}
			return cfg, nil
		}
		return Config{}, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.BindAddress == "" {
		cfg.BindAddress = defaultBindAddress
	}
	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = filepath.Join(binaryDir, defaultDatabaseName)
	} else if !filepath.IsAbs(cfg.DatabasePath) {
		cfg.DatabasePath = filepath.Join(binaryDir, cfg.DatabasePath)
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("invalid port %d", cfg.Port)
	}

	return cfg, nil
}

func writeDefaultConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(path, data, 0600)
}

func (c Config) ServerAddress() string {
	host := c.BindAddress
	if host == "*" {
		host = ""
	}
	return net.JoinHostPort(host, strconv.Itoa(c.Port))
}
