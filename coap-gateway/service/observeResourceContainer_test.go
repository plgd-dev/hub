package service

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_observeResourceContainer_Add(t *testing.T) {
	type args struct {
		observeResource observeResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				observeResource: observeResource{
					resourceID: "resourceID",
					remoteAddr: "remoteAddr",
					token:      []byte("token"),
				},
			},
		},
		{
			name: "ok-token",
			args: args{
				observeResource: observeResource{
					resourceID: "resourceID",
					remoteAddr: "remoteAddr",
					token:      []byte("token1"),
				},
			},
		},
		{
			name: "ok-remoteAddr",
			args: args{
				observeResource: observeResource{
					resourceID: "resourceID",
					remoteAddr: "remoteAddr1",
					token:      []byte("token"),
				},
			},
		},
		{
			name: "duplicit",
			args: args{
				observeResource: observeResource{
					resourceID: "resourceID",
					remoteAddr: "remoteAddr",
					token:      []byte("token"),
				},
			},
			wantErr: true,
		},
	}

	c := NewObserveResourceContainer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Add(tt.args.observeResource)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.Equal(t, 2, len(c.observersByRemoteAddr))
	assert.Equal(t, 1, len(c.observersByResource))
}

type sortObserveResource []*observeResource

func (s sortObserveResource) Len() int {
	return len(s)
}
func (s sortObserveResource) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortObserveResource) Less(i, j int) bool {
	if s[i].deviceID < s[j].deviceID {
		return true
	}
	if s[i].resourceID < s[j].resourceID {
		return true
	}
	if tokenToString(s[i].token) < tokenToString(s[j].token) {
		return true
	}
	return false
}

func Test_observeResourceContainer_Find(t *testing.T) {
	obs := []*observeResource{
		{
			resourceID: "a",
			token:      []byte("0"),
		},
		{
			resourceID: "a",
			token:      []byte("1"),
		},
		{
			resourceID: "b",
			token:      []byte("2"),
		},
	}

	type args struct {
		resourceID string
	}
	tests := []struct {
		name string
		args args
		want []*observeResource
	}{
		{
			name: "found 1",
			args: args{
				resourceID: obs[2].resourceID,
			},
			want: []*observeResource{obs[2]},
		},
		{
			name: "found 2",
			args: args{
				resourceID: obs[0].resourceID,
			},
			want: obs[:2],
		},
		{
			name: "not found",
			args: args{
				resourceID: "not found",
			},
			want: []*observeResource{},
		},
	}

	c := NewObserveResourceContainer()
	for _, ob := range obs {
		err := c.Add(*ob)
		assert.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Find(tt.args.resourceID)
			sort.Sort(sortObserveResource(tt.want))
			sort.Sort(sortObserveResource(got))
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_observeResourceContainer_RemoveByResource(t *testing.T) {
	obs := []observeResource{
		{
			resourceID: "a",
			token:      []byte("0"),
		},
		{
			resourceID: "a",
			token:      []byte("1"),
		},
		{
			resourceID: "b",
			token:      []byte("2"),
		},
	}
	type args struct {
		resourceID string
		remoteAddr string
		token      []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "remove0",
			args: args{
				resourceID: obs[2].resourceID,
				token:      obs[2].token,
			},
		},
		{
			name: "not found",
			args: args{
				resourceID: obs[2].resourceID,
				token:      obs[2].token,
			},
			wantErr: true,
		},
		{
			name: "remove1",
			args: args{
				resourceID: obs[1].resourceID,
				token:      obs[1].token,
			},
		},
		{
			name: "remove2",
			args: args{
				resourceID: obs[0].resourceID,
				token:      obs[0].token,
			},
		},
	}

	c := NewObserveResourceContainer()
	for _, ob := range obs {
		err := c.Add(ob)
		assert.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.RemoveByResource(tt.args.resourceID, tt.args.remoteAddr, tt.args.token)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.Equal(t, 0, len(c.observersByRemoteAddr))
	assert.Equal(t, 0, len(c.observersByResource))
}

func Test_observeResourceContainer_PopByRemoteAddr(t *testing.T) {
	obs := []*observeResource{
		{
			resourceID: "a",
			remoteAddr: "A",
			token:      []byte("0"),
		},
		{
			resourceID: "a",
			remoteAddr: "A",
			token:      []byte("1"),
		},
		{
			resourceID: "b",
			remoteAddr: "B",
			token:      []byte("2"),
		},
	}

	type args struct {
		remoteAddr string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*observeResource
	}{
		{
			name: "remove 0",
			args: args{
				remoteAddr: obs[2].remoteAddr,
			},
			want: []*observeResource{obs[2]},
		},
		{
			name: "not found 0",
			args: args{
				remoteAddr: obs[2].remoteAddr,
			},
			wantErr: true,
		},
		{
			name: "remove1",
			args: args{
				remoteAddr: obs[1].remoteAddr,
			},
			want: []*observeResource{obs[0], obs[1]},
		},
		{
			name: "not found 1",
			args: args{
				remoteAddr: obs[0].remoteAddr,
			},
			wantErr: true,
		},
	}

	c := NewObserveResourceContainer()
	for _, ob := range obs {
		err := c.Add(*ob)
		assert.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.PopByRemoteAddr(tt.args.remoteAddr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			sort.Sort(sortObserveResource(tt.want))
			sort.Sort(sortObserveResource(got))
			assert.Equal(t, tt.want, got)
		})
	}

	assert.Equal(t, 0, len(c.observersByRemoteAddr))
	assert.Equal(t, 0, len(c.observersByResource))
}

func Test_observeResourceContainer_PopByRemoteAddrToken(t *testing.T) {
	obs := []*observeResource{
		{
			resourceID: "a",
			remoteAddr: "A",
			token:      []byte("0"),
		},
		{
			resourceID: "a",
			remoteAddr: "A",
			token:      []byte("1"),
		},
		{
			resourceID: "b",
			remoteAddr: "B",
			token:      []byte("2"),
		},
	}
	type args struct {
		remoteAddr string
		token      []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *observeResource
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				remoteAddr: obs[0].remoteAddr,
				token:      obs[0].token,
			},
			want: obs[0],
		},
		{
			name: "not found",
			args: args{
				remoteAddr: obs[0].remoteAddr,
				token:      obs[0].token,
			},
			wantErr: true,
		},
	}

	c := NewObserveResourceContainer()
	for _, ob := range obs {
		err := c.Add(*ob)
		assert.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.PopByRemoteAddrToken(tt.args.remoteAddr, tt.args.token)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}

	assert.Equal(t, 2, len(c.observersByRemoteAddr))
	assert.Equal(t, 2, len(c.observersByResource))
}
