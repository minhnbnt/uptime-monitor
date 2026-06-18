package services

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/repository/monitor"
)

type mockUserRepo struct {
	findFn func(ctx context.Context, id uint) (*domain.User, error)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	return m.findFn(ctx, id)
}

type mockEventRepo struct {
	getEnrichedFn func(ctx context.Context, userID uint, from, to time.Time) ([]monitorrepo.EnrichedEvent, error)
}

func (m *mockEventRepo) GetEnrichedEventsByUser(ctx context.Context, userID uint, from, to time.Time) ([]monitorrepo.EnrichedEvent, error) {
	return m.getEnrichedFn(ctx, userID, from, to)
}

type mockMailer struct {
	sendFn func(to, subject string, attachment io.Reader) error
}

func (m *mockMailer) Send(to, subject string, attachment io.Reader) error {
	return m.sendFn(to, subject, attachment)
}

func enrichedEvent(serverName, url string, status domain.ServerStatus) monitorrepo.EnrichedEvent {
	return monitorrepo.EnrichedEvent{
		ServerName: serverName,
		URL:        url,
		Status:     status,
		Time:       time.Now(),
	}
}

func TestSendReport_UserErrors(t *testing.T) {
	t.Run("user repo returns error", func(t *testing.T) {
		svc := &DigestService{
			logger: logger.NewMockLogger(),
			userRepo: &mockUserRepo{
				findFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return nil, io.ErrUnexpectedEOF
				},
			},
		}
		err := svc.SendReport(t.Context(), 1, time.Now())
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("user not found", func(t *testing.T) {
		svc := &DigestService{
			logger: logger.NewMockLogger(),
			userRepo: &mockUserRepo{
				findFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return nil, nil
				},
			},
		}
		err := svc.SendReport(t.Context(), 1, time.Now())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSendReport_DateRange(t *testing.T) {
	now := time.Now()

	t.Run("from date within 30 days uses original from", func(t *testing.T) {
		from := now.Add(-5 * 24 * time.Hour)
		var capturedFrom time.Time

		svc := &DigestService{
			logger: logger.NewMockLogger(),
			userRepo: &mockUserRepo{
				findFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return &domain.User{Email: "test@test.com"}, nil
				},
			},
			eventRepo: &mockEventRepo{
				getEnrichedFn: func(_ context.Context, _ uint, from, to time.Time) ([]monitorrepo.EnrichedEvent, error) {
					capturedFrom = from
					return nil, nil
				},
			},
			mailer: &mockMailer{
				sendFn: func(_, _ string, _ io.Reader) error {
					return nil
				},
			},
		}

		if err := svc.SendReport(t.Context(), 1, from); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedFrom.Sub(from) > time.Second {
			t.Errorf("from was clamped: capturedFrom=%v, original=%v", capturedFrom, from)
		}
	})

	t.Run("from date older than 30 days clamps to 30 days", func(t *testing.T) {
		from := now.Add(-60 * 24 * time.Hour)
		var capturedFrom time.Time

		svc := &DigestService{
			logger: logger.NewMockLogger(),
			userRepo: &mockUserRepo{
				findFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return &domain.User{Email: "test@test.com"}, nil
				},
			},
			eventRepo: &mockEventRepo{
				getEnrichedFn: func(_ context.Context, _ uint, from, to time.Time) ([]monitorrepo.EnrichedEvent, error) {
					capturedFrom = from
					return nil, nil
				},
			},
			mailer: &mockMailer{
				sendFn: func(_, _ string, _ io.Reader) error {
					return nil
				},
			},
		}

		if err := svc.SendReport(t.Context(), 1, from); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedFrom := now.Add(-maxDigestRange)
		diff := capturedFrom.Sub(expectedFrom)
		if diff < 0 {
			diff = -diff
		}
		if diff > time.Second {
			t.Errorf("from was not clamped to 30 days: capturedFrom=%v, expected~=%v", capturedFrom, expectedFrom)
		}
	})

	t.Run("from date exactly 30 days ago is not clamped", func(t *testing.T) {
		from := now.Add(-30 * 24 * time.Hour)
		var capturedFrom time.Time

		svc := &DigestService{
			logger: logger.NewMockLogger(),
			userRepo: &mockUserRepo{
				findFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return &domain.User{Email: "test@test.com"}, nil
				},
			},
			eventRepo: &mockEventRepo{
				getEnrichedFn: func(_ context.Context, _ uint, from, to time.Time) ([]monitorrepo.EnrichedEvent, error) {
					capturedFrom = from
					return nil, nil
				},
			},
			mailer: &mockMailer{
				sendFn: func(_, _ string, _ io.Reader) error {
					return nil
				},
			},
		}

		if err := svc.SendReport(t.Context(), 1, from); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedFrom.Sub(from) > time.Second {
			t.Errorf("from was clamped: capturedFrom=%v, original=%v", capturedFrom, from)
		}
	})
}

func TestSendReport_EventErrors(t *testing.T) {
	t.Run("event repo error returns error", func(t *testing.T) {
		svc := &DigestService{
			logger: logger.NewMockLogger(),
			userRepo: &mockUserRepo{
				findFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return &domain.User{Email: "test@test.com"}, nil
				},
			},
			eventRepo: &mockEventRepo{
				getEnrichedFn: func(_ context.Context, _ uint, _, _ time.Time) ([]monitorrepo.EnrichedEvent, error) {
					return nil, io.ErrClosedPipe
				},
			},
		}

		err := svc.SendReport(t.Context(), 1, time.Now())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSendReport_WithEvents(t *testing.T) {
	now := time.Now()
	events := []monitorrepo.EnrichedEvent{
		enrichedEvent("server-a", "https://example.com/a", domain.StatusOn),
		enrichedEvent("server-b", "https://example.com/b", domain.StatusOff),
	}

	t.Run("sends mail with user email and events", func(t *testing.T) {
		var capturedTo, capturedSubject string

		svc := &DigestService{
			logger: logger.NewMockLogger(),
			userRepo: &mockUserRepo{
				findFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return &domain.User{Email: "admin@example.com"}, nil
				},
			},
			eventRepo: &mockEventRepo{
				getEnrichedFn: func(_ context.Context, _ uint, _, _ time.Time) ([]monitorrepo.EnrichedEvent, error) {
					return events, nil
				},
			},
			mailer: &mockMailer{
				sendFn: func(to, subject string, _ io.Reader) error {
					capturedTo = to
					capturedSubject = subject
					return nil
				},
			},
		}

		if err := svc.SendReport(t.Context(), 1, now.Add(-7*24*time.Hour)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTo != "admin@example.com" {
			t.Errorf("to = %q, want admin@example.com", capturedTo)
		}
		if capturedSubject != "Uptime Monitor - Daily Digest - "+now.Format("2006-01-02") {
			t.Errorf("subject = %q", capturedSubject)
		}
	})

	t.Run("empty events sends empty report", func(t *testing.T) {
		var sendCalled bool

		svc := &DigestService{
			logger: logger.NewMockLogger(),
			userRepo: &mockUserRepo{
				findFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return &domain.User{Email: "empty@example.com"}, nil
				},
			},
			eventRepo: &mockEventRepo{
				getEnrichedFn: func(_ context.Context, _ uint, _, _ time.Time) ([]monitorrepo.EnrichedEvent, error) {
					return nil, nil
				},
			},
			mailer: &mockMailer{
				sendFn: func(_, _ string, _ io.Reader) error {
					sendCalled = true
					return nil
				},
			},
		}

		if err := svc.SendReport(t.Context(), 1, now.Add(-7*24*time.Hour)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !sendCalled {
			t.Error("mailer.Send was not called")
		}
	})

	t.Run("mailer error returns error", func(t *testing.T) {
		svc := &DigestService{
			logger: logger.NewMockLogger(),
			userRepo: &mockUserRepo{
				findFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return &domain.User{Email: "test@test.com"}, nil
				},
			},
			eventRepo: &mockEventRepo{
				getEnrichedFn: func(_ context.Context, _ uint, _, _ time.Time) ([]monitorrepo.EnrichedEvent, error) {
					return events, nil
				},
			},
			mailer: &mockMailer{
				sendFn: func(_, _ string, _ io.Reader) error {
					return io.ErrShortWrite
				},
			},
		}

		err := svc.SendReport(t.Context(), 1, time.Now())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
