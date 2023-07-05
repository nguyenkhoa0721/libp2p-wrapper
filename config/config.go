package config

import (
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	P2P P2P
}

func LoadConfig(path string) (*Config, error) {
	//viper.AddConfigPath(".")
	//viper.SetConfigName("config")
	//viper.SetConfigType("yml")
	//viper.AutomaticEnv()

	viper.SetConfigFile(path)

	err := viper.ReadInConfig()
	if err != nil {
		logrus.Error(err)
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		config = nil
		return config, err
	}
	config.P2P.Dests = strings.Split(config.P2P.Dest, ",")

	return config, nil
}
