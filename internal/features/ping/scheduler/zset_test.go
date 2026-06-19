package scheduler

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestGetScheduledTask(t *testing.T) {
	tests := []struct {
		name    string
		member  any
		score   any
		want    *ScheduledTask
		wantErr bool
	}{
		{
			name:   "valid inputs",
			member: "42",
			score:  "1000000",
			want: &ScheduledTask{
				EndpointID: 42,
				Score:      1000000,
			},
			wantErr: false,
		},
		{
			name:   "zero id",
			member: "0",
			score:  "0",
			want: &ScheduledTask{
				EndpointID: 0,
				Score:      0,
			},
			wantErr: false,
		},
		{
			name:   "negative score",
			member: "1",
			score:  "-500",
			want: &ScheduledTask{
				EndpointID: 1,
				Score:      -500,
			},
			wantErr: false,
		},
		{
			name:   "large values",
			member: "999999",
			score:  "9999999999999",
			want: &ScheduledTask{
				EndpointID: 999999,
				Score:      9999999999999,
			},
			wantErr: false,
		},
		{
			name:    "member not a string",
			member:  42,
			score:   "1000",
			wantErr: true,
		},
		{
			name:    "score not a string",
			member:  "1",
			score:   1000,
			wantErr: true,
		},
		{
			name:    "member not a valid uint",
			member:  "abc",
			score:   "1000",
			wantErr: true,
		},
		{
			name:    "score not a valid int64",
			member:  "1",
			score:   "notanumber",
			wantErr: true,
		},
		{
			name:    "negative member string",
			member:  "-1",
			score:   "1000",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getScheduledTask(tt.member, tt.score)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.EndpointID != tt.want.EndpointID {
				t.Errorf("EndpointID = %d, want %d", got.EndpointID, tt.want.EndpointID)
			}
			if got.Score != tt.want.Score {
				t.Errorf("Score = %d, want %d", got.Score, tt.want.Score)
			}
		})
	}
}

func TestCollectScheduledTask(t *testing.T) {
	tests := []struct {
		name        string
		cmdVal      any
		wantDue     []ScheduledTask
		wantNext    ScheduledTask
		wantHasNext bool
		wantErr     bool
	}{
		{
			name: "due and next both populated",
			cmdVal: []any{
				[]any{"1", "100", "2", "200"},
				[]any{"3", "300"},
			},
			wantDue: []ScheduledTask{
				{EndpointID: 1, Score: 100},
				{EndpointID: 2, Score: 200},
			},
			wantNext:    ScheduledTask{EndpointID: 3, Score: 300},
			wantHasNext: true,
		},
		{
			name: "due populated, no next",
			cmdVal: []any{
				[]any{"10", "1000"},
				[]any{},
			},
			wantDue: []ScheduledTask{
				{EndpointID: 10, Score: 1000},
			},
			wantHasNext: false,
		},
		{
			name: "no due, no next",
			cmdVal: []any{
				[]any{},
				[]any{},
			},
			wantDue:     nil,
			wantHasNext: false,
		},
		{
			name: "no due, next populated",
			cmdVal: []any{
				[]any{},
				[]any{"5", "500"},
			},
			wantDue:     nil,
			wantNext:    ScheduledTask{EndpointID: 5, Score: 500},
			wantHasNext: true,
		},
		{
			name: "single element in due array",
			cmdVal: []any{
				[]any{"42", "4200"},
				[]any{},
			},
			wantDue: []ScheduledTask{
				{EndpointID: 42, Score: 4200},
			},
			wantHasNext: false,
		},
		{
			name:    "result not []any",
			cmdVal:  "invalid",
			wantErr: true,
		},
		{
			name: "wrong number of elements in result",
			cmdVal: []any{
				[]any{},
			},
			wantErr: true,
		},
		{
			name:    "dueRaw not []any",
			cmdVal:  []any{"string", []any{}},
			wantErr: true,
		},
		{
			name: "nextRaw not []any",
			cmdVal: []any{
				[]any{},
				"string",
			},
			wantErr: true,
		},
		{
			name: "odd number of elements in due",
			cmdVal: []any{
				[]any{"1", "100", "2"},
				[]any{},
			},
			wantDue: []ScheduledTask{
				{EndpointID: 1, Score: 100},
			},
			wantHasNext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &redis.Cmd{}
			cmd.SetVal(tt.cmdVal)

			due, next, hasNext, err := collectScheduledTask(cmd)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(due) != len(tt.wantDue) {
				t.Fatalf("due length = %d, want %d", len(due), len(tt.wantDue))
			}
			for i := range due {
				if due[i].EndpointID != tt.wantDue[i].EndpointID {
					t.Errorf("due[%d].EndpointID = %d, want %d", i, due[i].EndpointID, tt.wantDue[i].EndpointID)
				}
				if due[i].Score != tt.wantDue[i].Score {
					t.Errorf("due[%d].Score = %d, want %d", i, due[i].Score, tt.wantDue[i].Score)
				}
			}

			if hasNext != tt.wantHasNext {
				t.Errorf("hasNext = %v, want %v", hasNext, tt.wantHasNext)
			}
			if hasNext {
				if next.EndpointID != tt.wantNext.EndpointID {
					t.Errorf("next.EndpointID = %d, want %d", next.EndpointID, tt.wantNext.EndpointID)
				}
				if next.Score != tt.wantNext.Score {
					t.Errorf("next.Score = %d, want %d", next.Score, tt.wantNext.Score)
				}
			}
		})
	}
}

func TestSchedulerQueueKey(t *testing.T) {
	if schedulerQueueKey != "scheduler:queue" {
		t.Errorf("schedulerQueueKey = %q, want %q", schedulerQueueKey, "scheduler:queue")
	}
}

func TestClaimLock(t *testing.T) {
	if claimLock != 10*time.Second {
		t.Errorf("claimLock = %v, want %v", claimLock, 10*time.Second)
	}
}
