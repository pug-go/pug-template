package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Service struct {
		Name   string `yaml:"name" env:"SERVICE_NAME"`
		Domain string `yaml:"domain" env:"DOMAIN"`
		Ports  struct {
			Grpc  int16 `yaml:"grpc" env:"GRPC_PORT" env-default:"8080"`
			Http  int16 `yaml:"http" env:"HTTP_PORT" env-default:"8082"`
			Debug int16 `yaml:"debug" env:"DEBUG_PORT" env-default:"8084"`
		} `yaml:"ports"`
	} `yaml:"service"`
}

var GlobalConfig Config

func (c *Config) Load(path string) error {
	if path != "" {
		err := cleanenv.ReadConfig(path, c)
		if err != nil {
			return err
		}
		return nil
	}

	// if path not passed, read env only
	err := cleanenv.ReadEnv(c)
	if err != nil {
		return err
	}
	return nil
}
