package appstate

import "github.com/kkz6/launchctl/internal/config"

var cfg *config.Config

func SetConfig(c *config.Config) {
	cfg = c
}

func GetConfig() *config.Config {
	return cfg
}
