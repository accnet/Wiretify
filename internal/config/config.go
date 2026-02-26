package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ServerName     string `mapstructure:"SERVER_NAME"`
	InterfaceName  string `mapstructure:"WG_INTERFACE"`
	Port           int    `mapstructure:"WG_PORT"`
	Address        string `mapstructure:"WG_ADDRESS"`
	PrivateKey     string `mapstructure:"WG_PRIVATE_KEY"`
	ServerEndpoint string `mapstructure:"SERVER_ENDPOINT"`
	DatabasePath   string `mapstructure:"DB_PATH"`
	AdminPassword  string `mapstructure:"ADMIN_PASSWORD"`
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("SERVER_NAME", "Wiretify Server")
	viper.SetDefault("WG_INTERFACE", "wg0")
	viper.SetDefault("WG_PORT", 51820)
	viper.SetDefault("WG_ADDRESS", "10.8.0.1/24")
	viper.SetDefault("SERVER_ENDPOINT", "127.0.0.1")
	viper.SetDefault("DB_PATH", "wiretify.db")


	viper.SetEnvPrefix("WIRETIFY")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Support fallback loading from a file for private key if env is not set
	viper.SetConfigFile(".env")
	// Ignore err if .env doesn't exist
	_ = viper.ReadInConfig()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
