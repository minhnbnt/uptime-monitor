package service

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	ontimedto "github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
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
	servers      []domain.Server
	err          error
	totalCount   int64
	onlineCount  int64
	offlineCount int64
}

func (m *mockServerLister) List(_ context.Context, _ uint, _, _ int) ([]domain.Server, error) {
	return m.servers, m.err
}

func (m *mockServerLister) CountByStatus(_ context.Context, _ uint) (int64, int64, int64, error) {
	return m.totalCount, m.onlineCount, m.offlineCount, m.err
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
		serverRepo: &mockServerLister{servers: nil, totalCount: 1, onlineCount: 1, offlineCount: 0},
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

type mockNotificationConfigRepo struct {
	getByUserIDFn func(ctx context.Context, userID uint) (*domain.NotificationConfig, error)
}

func (m *mockNotificationConfigRepo) GetByUserID(ctx context.Context, userID uint) (*domain.NotificationConfig, error) {
	return m.getByUserIDFn(ctx, userID)
}

func TestBuildReport_Empty(t *testing.T) {
	svc := emptyDigestService()
	rows := svc.buildReport(nil, nil)
	if len(rows) != 0 {
		t.Errorf("got %d rows, want 0", len(rows))
	}
}

func TestBuildReport_SingleServer(t *testing.T) {
	svc := emptyDigestService()
	date := utils.TruncateDay(time.Now())
	s := domain.Server{Name: "Server A"}
	s.ID = 1
	rows := svc.buildReport([]domain.Server{s}, map[uint][]ontimedto.OntimeStats{
		1: {{Date: date, Stats: 99.5}},
	})
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].ServerID != 1 || rows[0].ServerName != "Server A" {
		t.Errorf("row = %+v", rows[0])
	}
	if rows[0].Stats[date] != 99.5 {
		t.Errorf("stats[date] = %f, want 99.5", rows[0].Stats[date])
	}
}

func TestBuildReport_MultipleServersSorted(t *testing.T) {
	svc := emptyDigestService()
	s1 := domain.Server{Name: "Zeta"}
	s1.ID = 1
	s2 := domain.Server{Name: "Alpha"}
	s2.ID = 2
	rows := svc.buildReport([]domain.Server{s1, s2}, nil)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].ServerName != "Alpha" || rows[1].ServerName != "Zeta" {
		t.Errorf("order: %q, %q", rows[0].ServerName, rows[1].ServerName)
	}
}

func TestSendUserDigest_UserNotFound(t *testing.T) {
	svc := emptyDigestService()
	svc.userRepo = &mockUserRepo{
		findFn: func(_ context.Context, _ uint) (*domain.User, error) {
			return nil, nil
		},
	}
	err := svc.SendUserDigest(t.Context(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendUserDigest_UserError(t *testing.T) {
	svc := emptyDigestService()
	svc.userRepo = &mockUserRepo{
		findFn: func(_ context.Context, _ uint) (*domain.User, error) {
			return nil, io.ErrUnexpectedEOF
		},
	}
	err := svc.SendUserDigest(t.Context(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendUserDigest_ConfigNotFound(t *testing.T) {
	svc := emptyDigestService()
	svc.userRepo = &mockUserRepo{
		findFn: func(_ context.Context, _ uint) (*domain.User, error) {
			return &domain.User{}, nil
		},
	}
	svc.configRepo = &mockNotificationConfigRepo{
		getByUserIDFn: func(_ context.Context, _ uint) (*domain.NotificationConfig, error) {
			return nil, nil
		},
	}
	err := svc.SendUserDigest(t.Context(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendUserDigest_ConfigInactive(t *testing.T) {
	svc := emptyDigestService()
	svc.userRepo = &mockUserRepo{
		findFn: func(_ context.Context, _ uint) (*domain.User, error) {
			return &domain.User{}, nil
		},
	}
	svc.configRepo = &mockNotificationConfigRepo{
		getByUserIDFn: func(_ context.Context, _ uint) (*domain.NotificationConfig, error) {
			return &domain.NotificationConfig{Active: false}, nil
		},
	}
	err := svc.SendUserDigest(t.Context(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendUserDigest_ConfigError(t *testing.T) {
	svc := emptyDigestService()
	svc.userRepo = &mockUserRepo{
		findFn: func(_ context.Context, _ uint) (*domain.User, error) {
			return &domain.User{}, nil
		},
	}
	svc.configRepo = &mockNotificationConfigRepo{
		getByUserIDFn: func(_ context.Context, _ uint) (*domain.NotificationConfig, error) {
			return nil, io.ErrClosedPipe
		},
	}
	err := svc.SendUserDigest(t.Context(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendUserDigest_Success(t *testing.T) {
	var capturedUserID uint
	svc := emptyDigestService()
	svc.userRepo = &mockUserRepo{
		findFn: func(_ context.Context, _ uint) (*domain.User, error) {
			return &domain.User{Email: "test@test.com"}, nil
		},
	}
	svc.configRepo = &mockNotificationConfigRepo{
		getByUserIDFn: func(_ context.Context, _ uint) (*domain.NotificationConfig, error) {
			return &domain.NotificationConfig{Active: true, FromDate: time.Now().Add(-7 * 24 * time.Hour)}, nil
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
	err := svc.SendUserDigest(t.Context(), capturedUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
