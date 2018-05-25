package config

import (
	"fmt"

	"github.com/jinzhu/configor"
)

type Config struct {
	APPName string `default:"ivorareteambot"`

	DB struct {
		Name     string
		User     string `default:"root"`
		Password string `required:"true" env:"example"`

		Host      string `default:"127.0.0.1"`
		Port      uint   `default:"3306"`
		Charset   string `default:"utf8"`
		ParseTime string `default:"true"`
	}
	SlackToken string `default:"someSlackToken"`
}

func GetConfig() Config {
	var config Config
	configor.Load(&config, "config/config.yml")
	fmt.Printf("config: %#v", config)

	return config
}