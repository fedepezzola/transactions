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

type EmailConfig struct {
	User     string `conf:"default:mail@gmail.com"`
	Password string `conf:"default:password,mask"`
	SmtpHost string `conf:"smtp.gmail.com"`
	SmtpPort string `conf:"587"`
	To       string `conf:"default:mail.recipient@gmail.com"`
}
type NotificationsConfig struct {
	Email EmailConfig
}

type AppConfig struct {
	conf.Version
	DB            DBConfig
	Notifications NotificationsConfig
	AccountNumber string `conf:"default:123456"`
	File          string `conf:"short:f"`
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
