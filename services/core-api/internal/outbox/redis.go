package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisPublisher struct {
	client redis.UniversalClient
	stream string
}

func NewRedisPublisher(client redis.UniversalClient, stream string) *RedisPublisher {
	if stream == "" {
		stream = "opportunity.events"
	}
	return &RedisPublisher{client: client, stream: stream}
}

func (p *RedisPublisher) Check(ctx context.Context) error {
	commands, err := p.client.Command(ctx).Result()
	if err != nil {
		return fmt.Errorf("inspect Redis commands: %w", err)
	}
	if commands["xadd"] == nil && commands["XADD"] == nil {
		return fmt.Errorf("Redis Streams requires XADD support (Redis 5 or newer)")
	}
	return nil
}

func (p *RedisPublisher) Publish(ctx context.Context, event Event) error {
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.stream,
		Values: map[string]any{
			"event_id": event.ID, "tenant_id": event.TenantID,
			"event_type": event.EventType, "event": string(encoded),
		},
	}).Err()
}
