package domain

import (
	"testing"
	"time"
)

func TestServerStatusConstants(t *testing.T) {
	if StatusOn != ServerStatus("ON") {
		t.Errorf("StatusOn = %q, want %q", StatusOn, "ON")
	}
	if StatusOff != ServerStatus("OFF") {
		t.Errorf("StatusOff = %q, want %q", StatusOff, "OFF")
	}
	if StatusOn == StatusOff {
		t.Error("StatusOn and StatusOff should be different")
	}
}

func TestServer_TableName(t *testing.T) {
	if (Server{}).TableName() != "servers" {
		t.Errorf("Server.TableName() = %q, want %q", (Server{}).TableName(), "servers")
	}
}

func TestEndpoint_TableName(t *testing.T) {
	if (Endpoint{}).TableName() != "endpoints" {
		t.Errorf("Endpoint.TableName() = %q, want %q", (Endpoint{}).TableName(), "endpoints")
	}
}

func TestServerEvent_TableName(t *testing.T) {
	if (ServerEvent{}).TableName() != "server_events" {
		t.Errorf("ServerEvent.TableName() = %q, want %q", (ServerEvent{}).TableName(), "server_events")
	}
}

func TestUser_TableName(t *testing.T) {
	if (User{}).TableName() != "users" {
		t.Errorf("User.TableName() = %q, want %q", (User{}).TableName(), "users")
	}
}

func TestServerFieldAccess(t *testing.T) {
	s := Server{Name: "test-server"}
	if s.Name != "test-server" {
		t.Errorf("Server.Name = %q, want %q", s.Name, "test-server")
	}
}

func TestEndpointFieldAccess(t *testing.T) {
	ep := Endpoint{
		ServerID: 1,
		URL:      "http://example.com",
	}
	if ep.URL != "http://example.com" {
		t.Errorf("Endpoint.URL = %q", ep.URL)
	}
}

func TestUserFieldAccess(t *testing.T) {
	u := User{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "hash",
		Name:     "Test User",
	}
	if u.Email != "test@example.com" {
		t.Errorf("User.Email = %q", u.Email)
	}
}

func TestServerEventFieldAccess(t *testing.T) {
	now := time.Now()
	se := ServerEvent{
		EndpointID: 1,
		Status:     StatusOn,
		Time:       now,
	}
	if se.Status != StatusOn {
		t.Errorf("ServerEvent.Status = %q, want %q", se.Status, StatusOn)
	}
}
