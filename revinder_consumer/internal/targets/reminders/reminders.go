package reminders

import (
	"context"
	"errors"

	"github.com/sottey/revinder_consumer/internal/bridge"
)

type Target struct{}

func New() *Target {
	return &Target{}
}

func (t *Target) Process(ctx context.Context, item bridge.Item) error {
	return errors.New("apple reminders target is not implemented")
}
