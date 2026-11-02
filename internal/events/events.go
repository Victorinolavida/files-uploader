package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const UploadedChannel = "files:uploaded"

type UploadedEvent struct {
	UserID      string    `json:"user_id"`
	FileID      string    `json:"file_id"`
	ObjectKey   string    `json:"object_key"`
	Name        string    `json:"name"`
	Size        int       `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

func PublishUploaded(ctx context.Context, rdb *redis.Client, ev UploadedEvent) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return rdb.Publish(ctx, UploadedChannel, payload).Err()
}

type Hub struct {
	rdb  *redis.Client
	mu   sync.RWMutex
	subs map[string]map[chan UploadedEvent]struct{}
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		rdb:  rdb,
		subs: make(map[string]map[chan UploadedEvent]struct{}),
	}
}

func (h *Hub) Run(ctx context.Context) error {
	pubsub := h.rdb.Subscribe(ctx, UploadedChannel)
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-pubsub.Channel():
			if !ok {
				return nil
			}
			var ev UploadedEvent
			if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
				slog.Error("invalid upload event", slog.Any("error", err))
				continue
			}
			h.broadcast(ev)
		}
	}
}

func (h *Hub) Subscribe(userID string) (<-chan UploadedEvent, func()) {
	ch := make(chan UploadedEvent, 16)

	h.mu.Lock()
	if h.subs[userID] == nil {
		h.subs[userID] = make(map[chan UploadedEvent]struct{})
	}
	h.subs[userID][ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		if subs := h.subs[userID]; subs != nil {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(h.subs, userID)
			}
		}
		h.mu.Unlock()
	}
}

func (h *Hub) broadcast(ev UploadedEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.subs[ev.UserID] {
		select {
		case ch <- ev:
		default:
		}
	}
}
