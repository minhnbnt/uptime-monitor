package service

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	ontimedto "github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type mockUserRepo struct {
	findFn func(ctx context.Context, id uint) (*domain.User, error)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	return m.findFn(ctx, id)
}

type mockMailer struct {
	sendFn func(to, subject string, attachment io.Reader) error
}

func (m *mockMailer) Send(to, subject string, attachment io.Reader) error {
	return m.sendFn(to, subject, attachment)
}

type mockServerLister struct {
	servers []domain.Server
	err     error
}

func (m *mockServerLister) List(_ context.Context, _ uint, _, _ int) ([]domain.Server, error) {
	return m.servers, m.err
}

type mockOntimeSvc struct {
	statsByServer map[uint][]ontimedto.OntimeStats
	err           error
}

func (m *mockOntimeSvc) GetServersOntimeForDates(_ context.Context, servers []domain.Server, dates []time.Time) (map[uint][]ontimedto.OntimeStats, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.statsByServer, nil
}

func emptyDigestService() *DigestService {
	return &DigestService{
		logger:     logger.NewMockLogger(),
		serverRepo: &mockServerLister{servers: nil},
	}
}

func TestSendReport_UserErrors(t *testing.T) {
	t.Run("user repo returns error", func(t *testing.T) {
		svc := emptyDigestService()
		svc.userRepo = &mockUserRepo{
			findFn: func(_ context.Context, _ uint) (*domain.User, error) {
				return nil, io.ErrUnexpectedEOF
			},
		}
		err := svc.SendReport(t.Context(), 1, time.Now())
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("user not found", func(t *testing.T) {
		svc := emptyDigestService()
		svc.userRepo = &mockUserRepo{
			findFn: func(_ context.Context, _ uint) (*domain.User, error) {
				return nil, nil
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

	t.Run("from date within 30 days returns report", func(t *testing.T) {
		from := now.Add(-5 * 24 * time.Hour)

		svc := emptyDigestService()
		svc.userRepo = &mockUserRepo{
			findFn: func(_ context.Context, _ uint) (*domain.User, error) {
				return &domain.User{Email: "test@test.com"}, nil
			},
		}
		svc.ontimeSvc = &mockOntimeSvc{
			statsByServer: make(map[uint][]ontimedto.OntimeStats),
		}
		svc.mailer = &mockMailer{
			sendFn: func(_, _ string, _ io.Reader) error {
				return nil
			},
		}

		if err := svc.SendReport(t.Context(), 1, from); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("from date older than 30 days clamps to 30 days", func(t *testing.T) {
		from := now.Add(-60 * 24 * time.Hour)

		svc := emptyDigestService()
		svc.userRepo = &mockUserRepo{
			findFn: func(_ context.Context, _ uint) (*domain.User, error) {
				return &domain.User{Email: "test@test.com"}, nil
			},
		}
		svc.ontimeSvc = &mockOntimeSvc{
			statsByServer: make(map[uint][]ontimedto.OntimeStats),
		}
		svc.mailer = &mockMailer{
			sendFn: func(_, _ string, _ io.Reader) error {
				return nil
			},
		}

		if err := svc.SendReport(t.Context(), 1, from); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("from date exactly 30 days ago is not clamped", func(t *testing.T) {
		from := now.Add(-30 * 24 * time.Hour)

		svc := emptyDigestService()
		svc.userRepo = &mockUserRepo{
			findFn: func(_ context.Context, _ uint) (*domain.User, error) {
				return &domain.User{Email: "test@test.com"}, nil
			},
		}
		svc.ontimeSvc = &mockOntimeSvc{
			statsByServer: make(map[uint][]ontimedto.OntimeStats),
		}
		svc.mailer = &mockMailer{
			sendFn: func(_, _ string, _ io.Reader) error {
				return nil
			},
		}

		if err := svc.SendReport(t.Context(), 1, from); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSendReport_OntimeServiceError(t *testing.T) {
	svc := emptyDigestService()
	svc.userRepo = &mockUserRepo{
		findFn: func(_ context.Context, _ uint) (*domain.User, error) {
			return &domain.User{Email: "test@test.com"}, nil
		},
	}
	svc.ontimeSvc = &mockOntimeSvc{
		err: io.ErrClosedPipe,
	}

	err := svc.SendReport(t.Context(), 1, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendReport_ServerError(t *testing.T) {
	svc := emptyDigestService()
	svc.serverRepo = &mockServerLister{err: io.ErrUnexpectedEOF}
	svc.userRepo = &mockUserRepo{
		findFn: func(_ context.Context, _ uint) (*domain.User, error) {
			return &domain.User{Email: "test@test.com"}, nil
		},
	}

	err := svc.SendReport(t.Context(), 1, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendReport_SendsMail(t *testing.T) {
	now := time.Now()

	t.Run("sends mail with user email and report", func(t *testing.T) {
		var capturedTo, capturedSubject string

		svc := emptyDigestService()
		svc.userRepo = &mockUserRepo{
			findFn: func(_ context.Context, _ uint) (*domain.User, error) {
				return &domain.User{Email: "admin@example.com"}, nil
			},
		}
		svc.ontimeSvc = &mockOntimeSvc{
			statsByServer: make(map[uint][]ontimedto.OntimeStats),
		}
		svc.mailer = &mockMailer{
			sendFn: func(to, subject string, _ io.Reader) error {
				capturedTo = to
				capturedSubject = subject
				return nil
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

	t.Run("no servers sends empty report", func(t *testing.T) {
		var sendCalled bool

		svc := emptyDigestService()
		svc.userRepo = &mockUserRepo{
			findFn: func(_ context.Context, _ uint) (*domain.User, error) {
				return &domain.User{Email: "empty@example.com"}, nil
			},
		}
		svc.ontimeSvc = &mockOntimeSvc{
			statsByServer: make(map[uint][]ontimedto.OntimeStats),
		}
		svc.mailer = &mockMailer{
			sendFn: func(_, _ string, _ io.Reader) error {
				sendCalled = true
				return nil
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
		svc := emptyDigestService()
		svc.userRepo = &mockUserRepo{
			findFn: func(_ context.Context, _ uint) (*domain.User, error) {
				return &domain.User{Email: "test@test.com"}, nil
			},
		}
		svc.ontimeSvc = &mockOntimeSvc{
			statsByServer: make(map[uint][]ontimedto.OntimeStats),
		}
		svc.mailer = &mockMailer{
			sendFn: func(_, _ string, _ io.Reader) error {
				return io.ErrShortWrite
			},
		}

		err := svc.SendReport(t.Context(), 1, time.Now())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
