package infrastructure

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/samber/do/v2"
	gomail "github.com/wneessen/go-mail"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/service"
)

type Mailer struct {
	mailClient  *gomail.Client
	fromAddress string
	logger      *slog.Logger
}

func RegisterMailer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (service.MailSender, error) {

		cfg := do.MustInvoke[*config.Config](i)
		mailClientWrapper := do.MustInvoke[*config.MailClientWrapper](i)

		return &Mailer{
			mailClient:  mailClientWrapper.GetClient(),
			fromAddress: cfg.Mail.FromAddress,
			logger:      do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (m *Mailer) Send(to, subject string, attachment io.Reader) error {

	m.logger.Debug(
		"mailer.Send: preparing email",
		slog.String("to", to),
		slog.String("subject", subject),
	)

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

		m.logger.Error(
			"mailer.Send: failed to send",
			slog.String("to", to),
			slog.String("subject", subject),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to send mail: %w", err)
	}

	m.logger.Info(
		"mailer.Send: email sent",
		slog.String("to", to),
		slog.String("subject", subject),
	)

	return nil
}
