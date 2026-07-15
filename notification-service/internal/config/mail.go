package config

import (
	"fmt"
	"sync"

	"github.com/samber/do/v2"
	gomail "github.com/wneessen/go-mail"
)

type MailClientWrapper struct {
	client *gomail.Client
	mu     sync.Mutex
}

func (w *MailClientWrapper) GetClient() *gomail.Client {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.client
}

func RegisterMailClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*MailClientWrapper, error) {
		cfg := do.MustInvoke[*Config](i)

		c, err := gomail.NewClient(
			cfg.Mail.Host,
			gomail.WithPort(cfg.Mail.Port),
			gomail.WithUsername(cfg.Mail.Username),
			gomail.WithPassword(cfg.Mail.Password),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create mail client: %w", err)
		}

		return &MailClientWrapper{client: c}, nil
	})
}
