package config

import (
	"github.com/ardanlabs/conf/v3"
)

type DBConfig struct {
	User         string `conf:"default:postgres"`
	Password     string `conf:"default:postgres,mask"`
	Host         string `conf:"default:localhost"`
	Name         string `conf:"default:transactions"`
	MaxIdleConns int    `conf:"default:0"`
	MaxOpenConns int    `conf:"default:0"`
	DisableTLS   bool   `conf:"default:true"`
}

type AppConfig struct {
	conf.Version
	DB            DBConfig
	AccountNumber string `conf:"default:123456"`
}

func Parse(prefix string) (AppConfig, string, error) {
	cfg := AppConfig{
		Version: conf.Version{
			Desc: "copyright information here",
		},
	}

	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		//nolint:wrapcheck
		return cfg, help, err
	}

	return cfg, help, nil
}
