package consumer

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ErrPermanent = errors.New("permanent")

func permanent(err error) error {
	return errors.Join(ErrPermanent, err)
}

type debeziumServerData struct {
	ID          uint       `json:"id"`
	CreatedByID uuid.UUID  `json:"created_by_id"`
	DeletedAt   *time.Time `json:"deleted_at"`
}
type debeziumMessage struct {
	Before *debeziumServerData `json:"before"`
	After  *debeziumServerData `json:"after"`
	Op     string              `json:"op"`
}

func unmarshalDebeziumMessage(msg redis.XMessage) (debeziumMessage, error) {

	raw, ok := msg.Values["value"]
	if !ok {
		return debeziumMessage{}, permanent(errors.New("missing values field"))
	}

	rawStr, ok := raw.(string)
	if !ok {
		return debeziumMessage{}, permanent(errors.New("stream message value not string"))
	}

	event := debeziumMessage{}
	if err := json.Unmarshal([]byte(rawStr), &event); err != nil {
		return debeziumMessage{}, permanent(fmt.Errorf("stream message invalid json: %w", err))
	}

	return event, nil
}

func resolveServerID(event debeziumMessage) (uint, error) {

	if event.After != nil {
		return event.After.ID, nil
	}

	if event.Before != nil {
		return event.Before.ID, nil
	}

	return 0, permanent(errors.New("resolveServerID: event has no before or after"))
}
