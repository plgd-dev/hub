package events

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDevicesRegisteredSubject(t *testing.T) {
	type args struct {
		owner string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				owner: "a",
			},
			want: "plgd.owners.e1407479-3136-56c0-9908-bb02fb0339e2.registrations.devicesregistered",
		},
		{
			name: "*",
			args: args{
				owner: "*",
			},
			want: "plgd.owners.*.registrations.devicesregistered",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDevicesRegisteredSubject(tt.args.owner)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGetDevicesUnregisteredSubject(t *testing.T) {
	type args struct {
		owner string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				owner: "a",
			},
			want: "plgd.owners.e1407479-3136-56c0-9908-bb02fb0339e2.registrations.devicesunregistered",
		},
		{
			name: "*",
			args: args{
				owner: "*",
			},
			want: "plgd.owners.*.registrations.devicesunregistered",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDevicesUnregisteredSubject(tt.args.owner)
			require.Equal(t, tt.want, got)
		})
	}
}
