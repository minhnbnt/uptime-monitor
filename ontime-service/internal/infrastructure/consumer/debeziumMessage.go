package consumer

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrPermanent = errors.New("permanent")

func permanent(err error) error {
	return errors.Join(ErrPermanent, err)
}

type debeziumServerData struct {
	ID          uint       `json:"id"`
	CreatedByID uint       `json:"created_by_id"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type debeziumMessagePayload struct {
	Before *debeziumServerData `json:"before"`
	After  *debeziumServerData `json:"after"`
	Op     string              `json:"op"`
}

type debeziumMessage struct {
	Payload debeziumMessagePayload `json:"payload"`
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
