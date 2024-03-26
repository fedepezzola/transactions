package email

import (
	"bytes"
	"fmt"
	"net/smtp"
	"text/template"
	"time"

	"github.com/fedepezzola/transactions/foundation/config"
	"go.uber.org/zap"
)

type EmailNotificationListener struct {
	cfg config.EmailConfig
	log *zap.SugaredLogger
}

func NewEmailNotificationListener(cfg config.EmailConfig, log *zap.SugaredLogger) *EmailNotificationListener {
	return &EmailNotificationListener{
		cfg: cfg,
		log: log,
	}
}

func (e *EmailNotificationListener) Update(data any) error {
	return e.SendTemplatedEmail("/home/fedepezzola/transactions/infrastructure/notifications/email/transactions_email.html", "New transactions file processed.", data)
}

func (e *EmailNotificationListener) SendTemplatedEmail(templateName string, subject string, data any) error {
	// Receiver email address.
	to := []string{
		e.cfg.To,
	}

	// Authentication.
	auth := smtp.PlainAuth("", e.cfg.User, e.cfg.Password, e.cfg.SmtpHost)

	funcMap := template.FuncMap{
		"add": func(a int, b int) int {
			return a + b
		},
		"month": func(a int) string {
			return time.Month(a).String()
		},
	}

	t, err := template.New("transactions_email.html").Funcs(funcMap).ParseFiles(templateName)
	if err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	var body bytes.Buffer

	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: %s \n%s\n\n", subject, mimeHeaders)))

	err = t.Execute(&body, data)
	if err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	// Sending email.
	err = smtp.SendMail(e.cfg.SmtpHost+":"+e.cfg.SmtpPort, auth, e.cfg.User, to, body.Bytes())
	if err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	e.log.Infof("Email with subject %s sent to %s", subject, e.cfg.To)
	return nil
}
