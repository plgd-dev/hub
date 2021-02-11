package eventbus

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockEventHandler struct {
	events    []EventUnmarshaler
	processed chan struct{}
}

func newMockEventHandler() *mockEventHandler {
	return &mockEventHandler{events: make([]EventUnmarshaler, 0, 10), processed: make(chan struct{})}
}

type eventUnmarshaler struct {
	version     uint64
	eventType   string
	aggregateID string
	groupID     string
}

func (e *eventUnmarshaler) Version() uint64 {
	return e.version
}
func (e *eventUnmarshaler) EventType() string {
	return e.eventType
}
func (e *eventUnmarshaler) AggregateID() string {
	return e.aggregateID
}
func (e *eventUnmarshaler) GroupID() string {
	return e.groupID
}
func (e *eventUnmarshaler) Unmarshal(v interface{}) error {
	return nil
}

func (eh *mockEventHandler) Handle(ctx context.Context, iter Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return errors.New("cannot determine type of event")
		}
		eh.events = append(eh.events, eu)
	}
	close(eh.processed)

	return iter.Err()
}

func TestGoroutinePoolHandler_Handle(t *testing.T) {
	type fields struct {
		goroutinePoolGo GoroutinePoolGoFunc
	}
	type args struct {
		ctx  context.Context
		iter Iter
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		wantTimeout bool
		want        []EventUnmarshaler
	}{
		{
			name: "empty",
			fields: fields{
				goroutinePoolGo: func(f func()) error { go f(); return nil },
			},
			args: args{
				ctx:  context.Background(),
				iter: &iter{},
			},
			wantTimeout: true,
		},
		{
			name: "valid",
			fields: fields{
				goroutinePoolGo: func(f func()) error { go f(); return nil },
			},
			args: args{
				ctx: context.Background(),
				iter: &iter{
					events: []EventUnmarshaler{
						&eventUnmarshaler{
							version:   0,
							eventType: "type-0",
						},
						&eventUnmarshaler{
							version:   1,
							eventType: "type-1",
						},
						&eventUnmarshaler{
							version:   2,
							eventType: "type-2",
						},
					},
				},
			},
			want: []EventUnmarshaler{
				&eventUnmarshaler{
					version:   0,
					eventType: "type-0",
				},
				&eventUnmarshaler{
					version:   1,
					eventType: "type-1",
				},
				&eventUnmarshaler{
					version:   2,
					eventType: "type-2",
				},
			},
		},
		{
			name: "error",
			fields: fields{
				goroutinePoolGo: func(f func()) error { go f(); return nil },
			},
			args: args{
				ctx: context.Background(),
				iter: &iter{
					events: []EventUnmarshaler{
						&eventUnmarshaler{
							version:   0,
							eventType: "type-0",
						},
						&eventUnmarshaler{
							version: 1,
						},
					},
				},
			},
			want: []EventUnmarshaler{
				&eventUnmarshaler{
					version:   0,
					eventType: "type-0",
				},
			},
			wantErr:     true,
			wantTimeout: true,
		},
		{
			name: "valid without goroutinePoolGo",
			args: args{
				ctx: context.Background(),
				iter: &iter{
					events: []EventUnmarshaler{
						&eventUnmarshaler{
							version:   0,
							eventType: "type-0",
						},
						&eventUnmarshaler{
							version:   1,
							eventType: "type-1",
						},
						&eventUnmarshaler{
							version:   2,
							eventType: "type-2",
						},
					},
				},
			},
			want: []EventUnmarshaler{
				&eventUnmarshaler{
					version:   0,
					eventType: "type-0",
				},
				&eventUnmarshaler{
					version:   1,
					eventType: "type-1",
				},
				&eventUnmarshaler{
					version:   2,
					eventType: "type-2",
				},
			},
		},
		{
			name: "error without goroutinePoolGo",
			fields: fields{
				goroutinePoolGo: func(f func()) error { go f(); return nil },
			},
			args: args{
				ctx: context.Background(),
				iter: &iter{
					events: []EventUnmarshaler{
						&eventUnmarshaler{
							version:   0,
							eventType: "type-0",
						},
						&eventUnmarshaler{
							version: 1,
						},
					},
				},
			},
			want: []EventUnmarshaler{
				&eventUnmarshaler{
					version:   0,
					eventType: "type-0",
				},
			},
			wantErr:     true,
			wantTimeout: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := newMockEventHandler()
			wantErr := tt.wantErr
			ep := NewGoroutinePoolHandler(
				tt.fields.goroutinePoolGo,
				eh,
				func(err error) {
					if wantErr {
						assert.Error(t, err)
					} else {
						assert.NoError(t, err)
					}
				})
			err := ep.Handle(tt.args.ctx, tt.args.iter)
			assert.NoError(t, err)
			select {
			case <-eh.processed:
				assert.Equal(t, tt.want, eh.events)
			case <-time.After(time.Millisecond * 100):
				if !tt.wantTimeout {
					assert.NoError(t, fmt.Errorf("timeout"))
				}
			}
		})
	}
}
