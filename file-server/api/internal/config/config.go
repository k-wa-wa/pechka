package config

import "github.com/caarlos0/env"

type Config struct {
	HlsResourceDir string `env:"HLS_RESOURCE_DIR" envDefault:"./resources/hls"`
}

func NewConfig() *Config {
	cfg := &Config{}
	env.Parse(cfg)
	return cfg
}
