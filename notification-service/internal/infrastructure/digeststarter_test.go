package infrastructure

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
)

const temporalImage = "temporalio/temporal:1.7.2"

var testClient temporalclient.Client
var testCleanup func()

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var err error
	testClient, testCleanup, err = startTemporalDevServer(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start temporal test server: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if testCleanup != nil {
		testCleanup()
	}

	os.Exit(code)
}

func startTemporalDevServer(ctx context.Context) (temporalclient.Client, func(), error) {
	req := tc.ContainerRequest{
		Image:        temporalImage,
		ExposedPorts: []string{"7233/tcp"},
		Cmd:          []string{"server", "start-dev", "--ip", "0.0.0.0", "--db-filename", "/home/temporal/temporal.db"},
		WaitingFor:   wait.ForListeningPort("7233/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("start temporal: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("get host: %w", err)
	}

	port, err := container.MappedPort(ctx, "7233/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("get port: %w", err)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())

	client, err := temporalclient.Dial(temporalclient.Options{
		HostPort: addr,
	})
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("dial temporal: %w", err)
	}

	cleanup := func() {
		client.Close()
		_ = container.Terminate(context.Background())
	}

	return client, cleanup, nil
}

func newTestDigestStarter(t *testing.T) *TemporalDigestStarter {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	return &TemporalDigestStarter{
		client:         testClient,
		scheduleClient: testClient.ScheduleClient(),
		taskQueue:      "test-queue",
		workflowName:   "test-workflow",
	}
}

func TestDigestStarter_UpsertAndDescribeSchedule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	starter := newTestDigestStarter(t)

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fromDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	err := starter.UpsertSchedule(t.Context(), userID, domain.ScheduleConfig{FromDate: fromDate, ToDate: toDate, DigestTime: "09:30"})
	require.NoError(t, err)

	info, err := starter.DescribeSchedule(t.Context(), userID)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.True(t, info.Exists)

	require.True(t, fromDate.Equal(info.FromDate), "expected %v, got %v", fromDate, info.FromDate)
	require.True(t, toDate.Equal(info.ToDate), "expected %v, got %v", toDate, info.ToDate)
	require.Equal(t, "09:30", info.DigestTime)
}

func TestDigestStarter_DescribeSchedule_NotExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	starter := newTestDigestStarter(t)

	info, err := starter.DescribeSchedule(t.Context(), uuid.MustParse("99999999-9999-9999-9999-999999999999"))
	require.NoError(t, err)
	require.NotNil(t, info)
	require.False(t, info.Exists)
}

func TestDigestStarter_DeleteSchedule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	starter := newTestDigestStarter(t)

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	fromDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	err := starter.UpsertSchedule(t.Context(), userID, domain.ScheduleConfig{FromDate: fromDate, ToDate: toDate, DigestTime: "08:00"})
	require.NoError(t, err)

	info, err := starter.DescribeSchedule(t.Context(), userID)
	require.NoError(t, err)
	require.True(t, info.Exists)

	err = starter.DeleteSchedule(t.Context(), userID)
	require.NoError(t, err)

	info, err = starter.DescribeSchedule(t.Context(), userID)
	require.NoError(t, err)
	require.False(t, info.Exists)
}

func TestDigestStarter_StartDigest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	starter := newTestDigestStarter(t)

	err := starter.StartDigest(t.Context(), uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	require.NoError(t, err)
}
