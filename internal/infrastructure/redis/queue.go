package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	r "github.com/redis/go-redis/v9"
)

const stream = "gdg:emails"
const group = "email-consumers"

type Queue struct {
	client      *r.Client
	claimCursor string
}

type Message struct {
	ID      string
	EmailID string
}

func Connect(redisURL string) (*r.Client, error) {
	options, err := r.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_URL")
	}
	options.DialTimeout = 5 * time.Second
	options.ReadTimeout = 5 * time.Second
	options.WriteTimeout = 5 * time.Second
	options.ContextTimeoutEnabled = true
	return r.NewClient(options), nil
}

func NewQueue(client *r.Client) *Queue {
	return &Queue{client: client, claimCursor: "0-0"}
}

func (q *Queue) Initialize(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (q *Queue) Publish(ctx context.Context, emailID string) error {
	return q.client.XAdd(ctx, &r.XAddArgs{Stream: stream, Values: map[string]any{"email_id": emailID}}).Err()
}

func (q *Queue) Receive(ctx context.Context, consumer string) (*Message, error) {
	pending, next, err := q.client.XAutoClaim(ctx, &r.XAutoClaimArgs{
		Stream: stream, Group: group, Consumer: consumer, MinIdle: 5 * time.Minute, Start: q.claimCursor, Count: 1,
	}).Result()
	if err != nil && !errors.Is(err, r.Nil) {
		return nil, err
	}
	if next != "" {
		q.claimCursor = next
	}
	if len(pending) > 0 {
		return message(pending[0])
	}
	streams, err := q.client.XReadGroup(ctx, &r.XReadGroupArgs{
		Group: group, Consumer: consumer, Streams: []string{stream, ">"}, Count: 1, Block: 2 * time.Second,
	}).Result()

	if errors.Is(err, r.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, nil
	}
	return message(streams[0].Messages[0])
}

func message(value r.XMessage) (*Message, error) {
	emailID, ok := value.Values["email_id"].(string)
	if !ok || emailID == "" {
		return nil, fmt.Errorf("invalid email queue entry %s", value.ID)
	}
	return &Message{ID: value.ID, EmailID: emailID}, nil
}

func (q *Queue) Ack(ctx context.Context, id string) error {
	// One consumer group owns this stream. Delete completed entries to bound storage.
	_, err := q.client.TxPipelined(ctx, func(pipe r.Pipeliner) error {
		pipe.XAck(ctx, stream, group, id)
		pipe.XDel(ctx, stream, id)
		return nil
	})
	return err
}
