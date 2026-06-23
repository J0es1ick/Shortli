package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort string   `mapstructure:"SERVER_PORT"`
	Database   Database `mapstructure:",squash"`
}

type Database struct {
	Host     string `mapstructure:"DATABASE_HOST"`
	Port     string `mapstructure:"DATABASE_PORT"`
	User     string `mapstructure:"DATABASE_USER"`
	Password string `mapstructure:"DATABASE_PASSWORD"`
	Name     string `mapstructure:"DATABASE_NAME"`
}

func InitConfig() (*Config, error) {
	exePath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	projectRoot := filepath.Dir(filepath.Dir(exePath))
	
	viper.SetConfigName(".env") 
	viper.SetConfigType("env")  
	viper.AddConfigPath(projectRoot)

	if err = viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}