package mongodb

import (
	"reflect"
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
)

func Test_getNormalizedGetEventsFilter(t *testing.T) {
	type args struct {
		queries []eventstore.GetEventsQuery
	}
	tests := []struct {
		name string
		args args
		want deviceIdFilter
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNormalizedGetEventsFilter(tt.args.queries); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getNormalizedGetEventsFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
