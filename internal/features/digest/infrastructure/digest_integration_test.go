package infrastructure

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	temporalclient "go.temporal.io/sdk/client"

	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var (
	temporalContainer testcontainers.Container
	temporalClient    temporalclient.Client
	digestStarter     *DigestStarter
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()

		container, address := testcontainers.StartTemporal(ctx)
		temporalContainer = container

		client, err := temporalclient.Dial(temporalclient.Options{
			HostPort: address,
		})
		if err != nil {
			log.Fatalf("create temporal client: %v", err)
		}
		temporalClient = client

		digestStarter = &DigestStarter{
			client:         client,
			scheduleClient: client.ScheduleClient(),
			taskQueue:      "test-digest-queue",
		}

		defer func() {
			client.Close()
			_ = container.Terminate(ctx)
		}()
	}

	os.Exit(m.Run())
}

func TestIntegration_Digest_StartDigest(t *testing.T) {
	testcontainers.SkipIfShort(t)

	ctx := t.Context()
	err := digestStarter.StartDigest(ctx, 42)
	if err != nil {
		t.Fatalf("StartDigest failed: %v", err)
	}
}

func TestIntegration_Digest_UpsertSchedule_Create(t *testing.T) {
	testcontainers.SkipIfShort(t)

	ctx := t.Context()
	now := time.Now()

	err := digestStarter.UpsertSchedule(ctx, 1, now, now.Add(24*time.Hour), "08:00")
	if err != nil {
		t.Fatalf("UpsertSchedule create failed: %v", err)
	}

	handle := digestStarter.scheduleClient.GetHandle(ctx, "digest-user-1")
	desc, err := handle.Describe(ctx)
	if err != nil {
		t.Fatalf("describe schedule after create: %v", err)
	}
	if desc == nil {
		t.Fatal("expected non-nil schedule description")
	}
}

func TestIntegration_Digest_UpsertSchedule_Update(t *testing.T) {
	testcontainers.SkipIfShort(t)

	ctx := t.Context()
	now := time.Now()

	err := digestStarter.UpsertSchedule(ctx, 2, now, now.Add(24*time.Hour), "08:00")
	if err != nil {
		t.Fatalf("UpsertSchedule create failed: %v", err)
	}

	err = digestStarter.UpsertSchedule(ctx, 2, now, now.Add(48*time.Hour), "10:00")
	if err != nil {
		t.Fatalf("UpsertSchedule update failed: %v", err)
	}

	handle := digestStarter.scheduleClient.GetHandle(ctx, "digest-user-2")
	desc, err := handle.Describe(ctx)
	if err != nil {
		t.Fatalf("describe schedule after update: %v", err)
	}
	if desc == nil {
		t.Fatal("expected non-nil schedule description")
	}
}

func TestIntegration_Digest_DeleteSchedule(t *testing.T) {
	testcontainers.SkipIfShort(t)

	ctx := t.Context()
	now := time.Now()

	err := digestStarter.UpsertSchedule(ctx, 3, now, now.Add(24*time.Hour), "08:00")
	if err != nil {
		t.Fatalf("UpsertSchedule create failed: %v", err)
	}

	err = digestStarter.DeleteSchedule(ctx, 3)
	if err != nil {
		t.Fatalf("DeleteSchedule failed: %v", err)
	}

	handle := digestStarter.scheduleClient.GetHandle(ctx, "digest-user-3")
	_, err = handle.Describe(ctx)
	if err == nil {
		t.Fatal("expected error describing deleted schedule")
	}
}
