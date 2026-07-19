package config

import (
	"crypto/tls"

	"github.com/samber/do/v2"
	gomail "github.com/wneessen/go-mail"
)

type MailClientWrapper struct {
	client *gomail.Client
}

func (m *MailClientWrapper) Shutdown() error {
	return m.client.Close()
}

func (m *MailClientWrapper) GetClient() *gomail.Client {
	return m.client
}

func getSecurityOption(config *MailConfig) []gomail.Option {

	if config.DisableSecurity {
		return []gomail.Option{
			gomail.WithTLSPolicy(gomail.TLSOpportunistic),
			gomail.WithSMTPAuth(gomail.SMTPAuthNoAuth),
		}
	}

	opts := []gomail.Option{
		gomail.WithTLSPolicy(gomail.TLSMandatory),
	}

	if config.SMTPUser != "" {

		opts = append(
			opts,
			gomail.WithUsername(config.SMTPUser),
			gomail.WithPassword(config.SMTPPassword),
			gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
		)

	} else {
		opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthNoAuth))
	}

	return opts
}

func newMailClient(config *MailConfig) (*gomail.Client, error) {

	opts := []gomail.Option{
		gomail.WithPort(config.SMTPPort),
	}

	opts = append(opts, getSecurityOption(config)...)

	if config.TLSInsecureSkipVerify {
		tlsConfig := tls.Config{InsecureSkipVerify: true}
		opts = append(opts, gomail.WithTLSConfig(&tlsConfig))
	}

	return gomail.NewClient(config.SMTPHost, opts...)
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
