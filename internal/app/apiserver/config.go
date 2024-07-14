package apiserver

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type AppConfig struct {
	BIND_ADDRESS string
	LOG_LEVEL    string
	FORWARD_URL  string
}

var config AppConfig

func init() {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	viper.SetConfigName(".env.local")
	if err := viper.MergeInConfig(); err != nil {
		log.Println(".env.local not found or could not be merged; using .env only")
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Error unmarshaling config: %s\n", err)
	}
	log.Println("Config loaded successfully")
}

func GetConfig() AppConfig {
	return config
}
