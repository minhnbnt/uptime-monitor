package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

var ErrPermanent = errors.New("permanent")

func permanent(err error) error {
	return errors.Join(ErrPermanent, err)
}

type debeziumMessage struct {
	Before *debeziumEndpointData `json:"before"`
	After  *debeziumEndpointData `json:"after"`
	Op     string                `json:"op"` // c=create, u=update, d=delete
}

type debeziumEndpointData struct {
	ID           uint   `json:"id"`
	URL          string `json:"url"`
	Method       string `json:"method"`
	ExpectedCode int    `json:"expected_code"`
	Interval     int64  `json:"interval"`
	Timeout      int64  `json:"timeout"`
}

func (d *debeziumEndpointData) toDomain() domain.Endpoint {
	return domain.Endpoint{
		ID:           d.ID,
		URL:          d.URL,
		Method:       d.Method,
		ExpectedCode: d.ExpectedCode,
		Interval:     time.Duration(d.Interval),
		Timeout:      time.Duration(d.Timeout),
	}
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
