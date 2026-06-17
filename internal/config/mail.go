package config

import (
	"github.com/samber/do/v2"
	gomail "github.com/wneessen/go-mail"
)

type MailConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUser     string `mapstructure:"smtp_user"`
	SMTPPassword string `mapstructure:"smtp_password"`
	FromAddress  string `mapstructure:"from_address"`
}

type MailClientWrapper struct {
	client *gomail.Client
}

func (m *MailClientWrapper) Shutdown() error {
	return m.client.Close()
}

func (m *MailClientWrapper) GetClient() *gomail.Client {
	return m.client
}

func newMailClient(config *MailConfig) (*gomail.Client, error) {
	return gomail.NewClient(
		config.SMTPHost,
		gomail.WithPort(config.SMTPPort),
		gomail.WithUsername(config.SMTPUser),
		gomail.WithPassword(config.SMTPPassword),
		gomail.WithTLSPolicy(gomail.TLSMandatory),
	)
}

func RegisterMailClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*MailClientWrapper, error) {

		config := do.MustInvoke[*Config](i)
		client, err := newMailClient(&config.Mail)

		if err != nil {
			return nil, err
		}

		return &MailClientWrapper{client: client}, nil
	})
}
