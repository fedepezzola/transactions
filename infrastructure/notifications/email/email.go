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
	return e.SendTemplatedEmail("New transactions file processed.", data)
}

func (e *EmailNotificationListener) SendTemplatedEmail(subject string, data any) error {
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

	t, err := template.New("transactions_email.html").Funcs(funcMap).Parse(templateEmail())
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

func templateEmail() string {
	return `
	<!DOCTYPE html>
	<html>
	<body>
		<img width="200" height="100" src='data:image/png;base64,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'/>
		<h3>New transactions file processed</h3><br/>
		<h3>Account Balance:</h3><span>{{.Balance}}</span><br/><br/>
		<table width="100%">
			<tr>
				<td width="50%">
					<span>Total balance is {{.FileBalance}}</span><br/>
					{{ range $i, $val := .TransactionsPerMonth}}
						{{if $val}}
							<span>Number of transactions in {{ month (add $i 1) }} :  {{$val}}</span> <br/>
						{{end}}
					{{end}}
				</td>
				<td width="50%" align="left">
					<span>Average Debit amount:  {{.DebitAvg}}</span><br/>
					<span>Average Credit amount:  {{.CreditAvg}}</span>
				</td>
			</tr>
		</div>
	</body>
	</html>
	`
}
