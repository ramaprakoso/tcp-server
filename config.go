package main

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config represents the application configuration

type Config struct {
	DB       DBConfig
	RabbitMQ RabbitMQConfig
	TCP      TCPConfig
	Redis    RedisConfig
}

type DBConfig struct {
	Username string
	Password string
	Host     string
	Port     string
	Name     string
}

type RabbitMQConfig struct {
	URL string
}

type TCPConfig struct {
	URL string
}

type RedisConfig struct {
	Address  string
	Password string
	DB       string
}

// LoadConfig loads the configuration from the file
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config") // name of your config file (without extension)
	viper.SetConfigType("yaml")   // config file type
	viper.AddConfigPath(".")      // optionally look for config in the working directory

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("Error unmarshaling config: %v", err)
	}

	return &config, nil
}
