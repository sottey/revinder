package consumer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/sottey/revinder/consumers/revinder_task_consumer/internal/bridge"
)

func TestProcessOnceProcessesTasksAndMarksProcessed(t *testing.T) {
	bridgeClient := &fakeBridge{
		items: []bridge.Item{
			{ID: 1, Type: "task", Title: "replace air filter"},
			{ID: 2, Type: "note", Title: "not a task"},
			{ID: 3, Type: "task", Title: "buy filters"},
		},
	}
	processor := &fakeTaskProcessor{}

	c := New(bridgeClient, processor, discardLogger())
	if err := c.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}

	if len(processor.items) != 2 {
		t.Fatalf("processed item count = %d, want 2", len(processor.items))
	}
	if processor.items[0].ID != 1 || processor.items[1].ID != 3 {
		t.Fatalf("processed item IDs = %v, want [1 3]", itemIDs(processor.items))
	}
	if len(bridgeClient.marked) != 2 {
		t.Fatalf("marked item count = %d, want 2", len(bridgeClient.marked))
	}
	if bridgeClient.marked[0] != 1 || bridgeClient.marked[1] != 3 {
		t.Fatalf("marked item IDs = %v, want [1 3]", bridgeClient.marked)
	}
	if len(bridgeClient.failed) != 0 {
		t.Fatalf("failed item IDs = %v, want none", bridgeClient.failed)
	}
}

func TestProcessOnceReturnsPendingItemsError(t *testing.T) {
	wantErr := errors.New("pending failed")
	c := New(&fakeBridge{pendingErr: wantErr}, &fakeTaskProcessor{}, discardLogger())

	err := c.ProcessOnce(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("ProcessOnce() error = %v, want %v", err, wantErr)
	}
}

func TestProcessOnceMarksFailedTaskProcessorFailed(t *testing.T) {
	bridgeClient := &fakeBridge{
		items: []bridge.Item{{ID: 1, Type: "task", Title: "replace air filter"}},
	}
	processor := &fakeTaskProcessor{err: errors.New("task failed")}

	c := New(bridgeClient, processor, discardLogger())
	if err := c.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}

	if len(processor.items) != 1 {
		t.Fatalf("processed item count = %d, want 1", len(processor.items))
	}
	if len(bridgeClient.marked) != 0 {
		t.Fatalf("marked item IDs = %v, want none", bridgeClient.marked)
	}
	if len(bridgeClient.failed) != 1 || bridgeClient.failed[0] != 1 {
		t.Fatalf("failed item IDs = %v, want [1]", bridgeClient.failed)
	}
}

func TestProcessOnceContinuesAfterMarkProcessedError(t *testing.T) {
	bridgeClient := &fakeBridge{
		items: []bridge.Item{
			{ID: 1, Type: "task", Title: "replace air filter"},
			{ID: 2, Type: "task", Title: "buy filters"},
		},
		markErrs: map[int64]error{
			1: errors.New("mark failed"),
		},
	}
	processor := &fakeTaskProcessor{}

	c := New(bridgeClient, processor, discardLogger())
	if err := c.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}

	if len(processor.items) != 2 {
		t.Fatalf("processed item count = %d, want 2", len(processor.items))
	}
	if len(bridgeClient.marked) != 2 {
		t.Fatalf("marked item count = %d, want 2", len(bridgeClient.marked))
	}
	if bridgeClient.marked[0] != 1 || bridgeClient.marked[1] != 2 {
		t.Fatalf("marked item IDs = %v, want [1 2]", bridgeClient.marked)
	}
}

type fakeBridge struct {
	items      []bridge.Item
	pendingErr error
	markErrs   map[int64]error
	failErrs   map[int64]error
	marked     []int64
	failed     []int64
}

func (f *fakeBridge) PendingItems(ctx context.Context) ([]bridge.Item, error) {
	if f.pendingErr != nil {
		return nil, f.pendingErr
	}
	return f.items, nil
}

func (f *fakeBridge) MarkProcessed(ctx context.Context, id int64) error {
	f.marked = append(f.marked, id)
	return f.markErrs[id]
}

func (f *fakeBridge) MarkFailed(ctx context.Context, id int64) error {
	f.failed = append(f.failed, id)
	return f.failErrs[id]
}

type fakeTaskProcessor struct {
	items []bridge.Item
	err   error
}

func (f *fakeTaskProcessor) Process(ctx context.Context, item bridge.Item) error {
	f.items = append(f.items, item)
	return f.err
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func itemIDs(items []bridge.Item) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
