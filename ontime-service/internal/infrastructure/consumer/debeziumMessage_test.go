package consumer

import (
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestPermanentClassification(t *testing.T) {

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"invalid json is permanent", permanent(errors.New("bad")), true},
		{"unexpected op is permanent", permanent(errors.New(`unexpected operation "z"`)), true},
		{"plain error is transient", errors.New("db down"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := errors.Is(tc.err, ErrPermanent); got != tc.want {
				t.Errorf("errors.Is(%v, ErrPermanent) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestUnmarshalDebeziumMessage(t *testing.T) {

	if _, err := unmarshalDebeziumMessage(redis.XMessage{}); !errors.Is(err, ErrPermanent) {
		t.Errorf("missing value: err = %v, want permanent", err)
	}

	if _, err := unmarshalDebeziumMessage(redis.XMessage{Values: map[string]any{"value": "not json"}}); !errors.Is(err, ErrPermanent) {
		t.Errorf("invalid json: err = %v, want permanent", err)
	}

	msg := redis.XMessage{Values: map[string]any{"value": `{"op":"c","after":{"id":1,"created_by_id":1}}`}}
	event, err := unmarshalDebeziumMessage(msg)
	if err != nil {
		t.Fatalf("valid message: unexpected err %v", err)
	}
	if event.Op != "c" || event.After == nil || event.After.ID != 1 {
		t.Errorf("valid message parsed incorrectly: %+v", event)
	}
}
