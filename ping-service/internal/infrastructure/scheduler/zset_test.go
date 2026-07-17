package scheduler

import (
	"testing"
)

func TestGetScheduledTask(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		task, err := getScheduledTask("42", "1000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.EndpointID != 42 {
			t.Errorf("EndpointID = %d, want 42", task.EndpointID)
		}
		if task.Score != 1000 {
			t.Errorf("Score = %d, want 1000", task.Score)
		}
	})

	t.Run("invalid member type", func(t *testing.T) {
		_, err := getScheduledTask(42, "1000")
		if err == nil {
			t.Fatal("expected error for int member")
		}
	})

	t.Run("invalid score type", func(t *testing.T) {
		_, err := getScheduledTask("42", 1000)
		if err == nil {
			t.Fatal("expected error for int score")
		}
	})

	t.Run("non-numeric member string", func(t *testing.T) {
		_, err := getScheduledTask("abc", "1000")
		if err == nil {
			t.Fatal("expected error for non-numeric member")
		}
	})

	t.Run("non-numeric score string", func(t *testing.T) {
		_, err := getScheduledTask("42", "not-a-number")
		if err == nil {
			t.Fatal("expected error for non-numeric score")
		}
	})
}

func TestClaimDueTasksZeroLimit(t *testing.T) {
	t.Run("zero limit returns nil due", func(t *testing.T) {
		r := &ZSetScheduleRepository{}
		due, next, hasNext, err := r.ClaimDueTasks(nil, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if due != nil {
			t.Errorf("due = %v, want nil", due)
		}
		if hasNext {
			t.Error("hasNext should be false")
		}
		if next != (ScheduledTask{}) {
			t.Errorf("next = %v, want zero value", next)
		}
	})
}
