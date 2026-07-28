package dto

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
)

func TestServerFromDomain(t *testing.T) {
	now := time.Now()

	t.Run("server with k8s fields", func(t *testing.T) {
		srv := domain.Server{
			Model: gorm.Model{
				ID:        42,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:          "Test Server",
			Namespace:     "prod",
			Kind:          "Deployment",
			ObjectID:      "web-app",
			ContainerName: "nginx",
			Interval:      30 * time.Second,
			Timeout:       10 * time.Second,
			CreatedByID:   1,
		}

		got := ServerFromDomain(srv)
		if got.ID != 42 {
			t.Errorf("ID = %d, want %d", got.ID, 42)
		}
		if got.Name != "Test Server" {
			t.Errorf("Name = %q, want %q", got.Name, "Test Server")
		}
		if got.Namespace != "prod" {
			t.Errorf("Namespace = %q, want %q", got.Namespace, "prod")
		}
		if got.Kind != "Deployment" {
			t.Errorf("Kind = %q, want %q", got.Kind, "Deployment")
		}
		if got.ObjectID != "web-app" {
			t.Errorf("ObjectID = %q, want %q", got.ObjectID, "web-app")
		}
		if got.ContainerName != "nginx" {
			t.Errorf("ContainerName = %q, want %q", got.ContainerName, "nginx")
		}
		if got.Interval != 30*time.Second {
			t.Errorf("Interval = %v, want %v", got.Interval, 30*time.Second)
		}
		if got.Timeout != 10*time.Second {
			t.Errorf("Timeout = %v, want %v", got.Timeout, 10*time.Second)
		}
		if !got.CreatedAt.Equal(now) {
			t.Errorf("CreatedAt mismatch")
		}
	})

	t.Run("maps timestamps correctly", func(t *testing.T) {
		createdAt := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)

		srv := domain.Server{
			Model: gorm.Model{
				ID:        1,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			Name:      "Timestamp Test",
			Namespace: "default",
			Kind:      "Pod",
			ObjectID:  "test-pod",
		}

		got := ServerFromDomain(srv)
		if !got.CreatedAt.Equal(createdAt) {
			t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, createdAt)
		}
		if !got.UpdatedAt.Equal(updatedAt) {
			t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, updatedAt)
		}
	})
}
