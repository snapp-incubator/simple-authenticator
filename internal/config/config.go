package config

import "github.com/spf13/viper"

type CustomConfig struct {
	WebserverConf WebserverConfig `mapstructure:"webserver"`
}

type WebserverConfig struct {
	Image         string `mapstrucutre:"image"`
	ContainerName string `mapstructure:"container_name"`
}

func InitConfig(configPath string) (*CustomConfig, error) {
	viper.AddConfigPath(configPath)
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	var customConfig CustomConfig
	err = viper.Unmarshal(&customConfig)
	if err != nil {
		return nil, err
	}
	return &customConfig, nil
}
