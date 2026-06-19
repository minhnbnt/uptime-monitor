package infrastructure

import (
	"fmt"
	"io"

	"github.com/samber/do/v2"
	gomail "github.com/wneessen/go-mail"

	"github.com/minhnbnt/uptime-monitor/internal/config"
)

type Mailer struct {
	mailClient  *gomail.Client
	fromAddress string
}

func RegisterMailer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Mailer, error) {

		cfg := do.MustInvoke[*config.Config](i)
		mailClientWrapper := do.MustInvoke[*config.MailClientWrapper](i)

		return &Mailer{
			mailClient:  mailClientWrapper.GetClient(),
			fromAddress: cfg.Mail.FromAddress,
		}, nil
	})
}

func (m *Mailer) Send(to, subject string, attachment io.Reader) error {

	msg := gomail.NewMsg()
	if err := msg.From(m.fromAddress); err != nil {
		return fmt.Errorf("failed to set from: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("failed to set to: %w", err)
	}

	msg.Subject(subject)
	if err := msg.AttachReader("report.xlsx", attachment); err != nil {
		return fmt.Errorf("failed to attach file: %w", err)
	}

	if err := m.mailClient.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send mail: %w", err)
	}

	return nil
}
