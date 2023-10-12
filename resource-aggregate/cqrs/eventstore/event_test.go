package eventstore

import (
	"testing"
	"time"
)

type mockEvent struct {
	aggregateID string
	groupID     string
	eventType   string
	version     uint64
	timestamp   time.Time
	etagData    *ETagData
	isSnapshot  bool
	serviceID   string
}

func (e *mockEvent) AggregateID() string {
	return e.aggregateID
}

func (e *mockEvent) GroupID() string {
	return e.groupID
}

func (e *mockEvent) EventType() string {
	return e.eventType
}

func (e *mockEvent) Version() uint64 {
	return e.version
}

func (e *mockEvent) Timestamp() time.Time {
	return e.timestamp
}

func (e *mockEvent) ServiceID() (string, bool) {
	if e.serviceID != "" {
		return e.serviceID, true
	}
	return "", false
}

func (e *mockEvent) ETag() *ETagData {
	return e.etagData
}

func (e *mockEvent) IsSnapshot() bool {
	return e.isSnapshot
}

func TestValidateEventsBeforeSave(t *testing.T) {
	ev := mockEvent{
		aggregateID: "d9e7e4a0-49b7-4e6e-8f00-9ebefb3f6f5d",
		groupID:     "d9e7e4a0-49b7-4e6e-8f00-9ebefb3f6f5d",
		eventType:   "event-type",
		version:     1,
		timestamp:   time.Now(),
	}
	type args struct {
		events []Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "empty-events",
			args:    args{},
			wantErr: true,
		},
		{
			name: "nil-events",
			args: args{
				events: []Event{
					nil,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-0-group-id",
			args: args{
				events: []Event{
					&mockEvent{
						aggregateID: ev.aggregateID,
						groupID:     "",
						eventType:   ev.eventType,
						version:     ev.version,
						timestamp:   time.Now(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-0-aggregate-id",
			args: args{
				events: []Event{
					&mockEvent{
						aggregateID: "",
						groupID:     ev.groupID,
						eventType:   ev.eventType,
						version:     ev.version,
						timestamp:   time.Now(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-0-timestamp",
			args: args{
				events: []Event{
					&mockEvent{
						aggregateID: ev.aggregateID,
						groupID:     ev.groupID,
						eventType:   ev.eventType,
						version:     ev.version,
						timestamp:   time.Time{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-1-group-id",
			args: args{
				events: []Event{
					&ev,
					&mockEvent{
						aggregateID: ev.aggregateID,
						groupID:     "",
						eventType:   ev.eventType,
						version:     ev.version + 1,
						timestamp:   time.Now(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-1-aggregate-id",
			args: args{
				events: []Event{
					&ev,
					&mockEvent{
						aggregateID: "",
						groupID:     ev.groupID,
						eventType:   ev.eventType,
						version:     ev.version + 1,
						timestamp:   time.Now(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-1-version",
			args: args{
				events: []Event{
					&ev,
					&mockEvent{
						aggregateID: ev.aggregateID,
						groupID:     ev.groupID,
						eventType:   ev.eventType,
						version:     ev.version,
						timestamp:   time.Now(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-1-event-type",
			args: args{
				events: []Event{
					&ev,
					&mockEvent{
						aggregateID: ev.aggregateID,
						groupID:     ev.groupID,
						eventType:   "",
						version:     ev.version + 1,
						timestamp:   time.Now(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-1-timestamp",
			args: args{
				events: []Event{
					&ev,
					&mockEvent{
						aggregateID: ev.aggregateID,
						groupID:     ev.groupID,
						eventType:   ev.eventType,
						version:     ev.version + 1,
						timestamp:   time.Time{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-1-service-id",
			args: args{
				events: []Event{
					&ev,
					&mockEvent{
						aggregateID: ev.aggregateID,
						groupID:     ev.groupID,
						eventType:   "event-type",
						serviceID:   "invalid-service-id",
						version:     ev.version + 1,
						timestamp:   time.Now(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-1-nil",
			args: args{
				events: []Event{
					&ev,
					nil,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-1-different-group-id",
			args: args{
				events: []Event{
					&ev,
					&mockEvent{
						aggregateID: ev.aggregateID,
						groupID:     "d9e7e4a0-49b7-4e6e-8f00-9ebefb3f6f5e",
						eventType:   ev.eventType,
						version:     ev.version + 1,
						timestamp:   time.Now(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-1-different-aggregate-id",
			args: args{
				events: []Event{
					&ev,
					&mockEvent{
						aggregateID: "d9e7e4a0-49b7-4e6e-8f00-9ebefb3f6f5e",
						groupID:     ev.groupID,
						eventType:   ev.eventType,
						version:     ev.version + 1,
						timestamp:   time.Now(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-event-1-timestamp-in-past",
			args: args{
				events: []Event{
					&ev,
					&mockEvent{
						aggregateID: ev.aggregateID,
						groupID:     ev.groupID,
						eventType:   ev.eventType,
						version:     ev.version + 1,
						timestamp:   ev.timestamp.Add(-time.Second),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid-event-1",
			args: args{
				events: []Event{
					&ev,
					&mockEvent{
						aggregateID: ev.aggregateID,
						groupID:     ev.groupID,
						eventType:   ev.eventType,
						version:     ev.version + 1,
						timestamp:   ev.timestamp.Add(time.Second),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateEventsBeforeSave(tt.args.events); (err != nil) != tt.wantErr {
				t.Errorf("ValidateEventsBeforeSave() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
