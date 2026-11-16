package state

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/bigelle/meshok/internal/database"
	"github.com/bigelle/meshok/internal/identity"
)

type State struct {
	DeviceId string
}

type StateLoader struct {
	store  StateStore
	logger *slog.Logger
}

func NewStateLoader(params LoaderParams) (*StateLoader, error) {
	if params.Store == nil {
		return nil, errors.New("config store is nil")
	}
	if params.Logger == nil {
		params.Logger = slog.Default()
	}

	return &StateLoader{
		store:  params.Store,
		logger: params.Logger,
	}, nil
}

type LoaderParams struct {
	Store  StateStore
	Logger *slog.Logger
}

type StateStore interface {
	GetAllStateValues(context.Context) ([]database.StateValue, error)
	StoreStateValue(context.Context, database.StateValue) error
}

func (l *StateLoader) LoadState(ctx context.Context) (*State, error) {
	state, err := l.store.GetAllStateValues(ctx)
	if err != nil {
		return nil, err
	}

	var st State
	// 1. if value is required but no fallback - err
	// 2. if value is required but there's a fallback - apply and store it
	// 3. if value is not required - skip
	for _, setting := range state {
		switch setting.Key {
		case "device_id":
			if setting.Value == "" {
				setting.Value = identity.GenerateRandomDeviceId()
				wctx, wcancel := context.WithTimeout(ctx, 1*time.Second)
				defer wcancel()

				if err := l.store.StoreStateValue(wctx, setting); err != nil {
					return nil, err
				}
			}
			st.DeviceId = setting.Value
		default:
			l.logger.Warn("unknown state field", "key", setting.Key)
		}
	}

	// check if the keys were even there:
	if st.DeviceId == "" {
		wctx, wcancel := context.WithTimeout(ctx, 1*time.Second)
		defer wcancel()
		id := identity.GenerateRandomDeviceId()
		if err := l.store.StoreStateValue(wctx, database.StateValue{
			Key:   "device_id",
			Value: id,
		}); err != nil {
			return nil, err
		}
		st.DeviceId = id
	}

	return &st, nil
}
