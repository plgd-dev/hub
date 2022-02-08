package queue_test

import (
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/sync/task/queue"
	"github.com/stretchr/testify/require"
)

type testArray struct {
	result []int
	mutex  sync.Mutex
}

func (a *testArray) append(i int) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.result = append(a.result, i)
}

func (a *testArray) reset() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.result = nil
}

func (a *testArray) copy() []int {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	b := make([]int, len(a.result))
	copy(b, a.result)
	return b
}

func TestQueue_SubmitForOneWorker(t *testing.T) {
	var result testArray

	type args struct {
		cfg            queue.Config
		key            interface{}
		preSharedTasks []func()
		tasks          []func()
		separateTasks  bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []int
	}{
		{
			name: "ok",
			args: args{
				cfg: queue.Config{
					GoPoolSize:  100,
					Size:        100,
					MaxIdleTime: time.Millisecond * 100,
				},
				key: 0,
				tasks: []func(){
					func() { result.reset() },
					func() { result.append(0) },
					func() { result.append(1) },
				},
			},
			want: []int{0, 1},
		},
		{
			name: "ok - separate tasks",
			args: args{
				cfg: queue.Config{
					GoPoolSize:  100,
					Size:        100,
					MaxIdleTime: time.Millisecond * 100,
				},
				key: 0,
				tasks: []func(){
					func() { result.reset() },
					func() { result.append(0) },
					func() { result.append(1) },
				},
				separateTasks: true,
			},
			want: []int{0, 1},
		},
		{
			name: "fail - separate tasks",
			args: args{
				cfg: queue.Config{
					GoPoolSize:  1,
					Size:        1,
					MaxIdleTime: time.Millisecond * 100,
				},
				key: 0,
				preSharedTasks: []func(){
					func() {},
				},
				tasks: []func(){
					func() { result.reset() },
					func() { result.append(0) },
				},
				separateTasks: true,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := queue.New(tt.args.cfg)
			require.NoError(t, err)
			defer q.Release()
			if len(tt.args.preSharedTasks) > 0 {
				err := q.Submit(tt.args.preSharedTasks...)
				require.NoError(t, err)
			}
			if tt.args.separateTasks {
				for _, task := range tt.args.tasks {
					err1 := q.SubmitForOneWorker(tt.args.key, task)
					if err1 != nil {
						err = err1
					}
				}
			} else {
				err = q.SubmitForOneWorker(tt.args.key, tt.args.tasks...)
			}
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			time.Sleep(time.Second * 3)
			require.Equal(t, tt.want, result.copy())
		})
	}
}
