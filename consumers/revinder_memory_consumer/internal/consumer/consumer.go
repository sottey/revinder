package consumer

import (
	"context"
	"log/slog"
	"time"

	"github.com/sottey/revinder/consumers/revinder_memory_consumer/internal/bridge"
)

type MemoryWriter interface {
	Write(ctx context.Context, item bridge.Item) error
}

type BridgeClient interface {
	PendingItems(ctx context.Context) ([]bridge.Item, error)
	MarkProcessed(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64) error
}

type Consumer struct {
	bridge       BridgeClient
	memoryWriter MemoryWriter
	logger       *slog.Logger
}

func New(bridgeClient BridgeClient, memoryWriter MemoryWriter, logger *slog.Logger) *Consumer {
	return &Consumer{
		bridge:       bridgeClient,
		memoryWriter: memoryWriter,
		logger:       logger,
	}
}

func (c *Consumer) Run(ctx context.Context, interval time.Duration) error {
	if err := c.ProcessOnce(ctx); err != nil {
		c.logger.Error("process_once_failed", "error", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.ProcessOnce(ctx); err != nil {
				c.logger.Error("process_once_failed", "error", err)
			}
		}
	}
}

func (c *Consumer) ProcessOnce(ctx context.Context) error {
	items, err := c.bridge.PendingItems(ctx)
	if err != nil {
		return err
	}
	c.logger.Info("pending_items_loaded", "count", len(items))

	for _, item := range items {
		if item.Type != "memory" {
			c.logger.Info("item_skipped", "id", item.ID, "type", item.Type)
			continue
		}

		if err := c.memoryWriter.Write(ctx, item); err != nil {
			c.logger.Error("item_write_failed", "id", item.ID, "error", err)
			if markErr := c.bridge.MarkFailed(ctx, item.ID); markErr != nil {
				c.logger.Error("item_mark_failed_failed", "id", item.ID, "error", markErr)
			}
			continue
		}

		if err := c.bridge.MarkProcessed(ctx, item.ID); err != nil {
			c.logger.Error("item_mark_processed_failed", "id", item.ID, "error", err)
			continue
		}

		c.logger.Info("item_processed", "id", item.ID)
	}

	return nil
}
