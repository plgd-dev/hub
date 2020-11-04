package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cqrs/event"
	"github.com/plgd-dev/cqrs/eventstore"
	"github.com/plgd-dev/cqrs/eventstore/maintenance"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
)

// Config represent application arguments
type Config struct {
	NumAggregates int    `long:"numAggregates" short:"n" default:"77" description:"a number of resource aggregates to perform cleanup onto"`
	BackupPath    string `long:"backupPath" short:"b" default:"/tmp/events.bkp" description:"backup text file path"`
	Mongo         mongodb.Config
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

type recordHandler struct {
	lock  sync.Mutex
	tasks []maintenance.Task
}

func newRecordHandler() *recordHandler {
	return &recordHandler{tasks: make([]maintenance.Task, 0, 77)}
}

func (eh *recordHandler) SetElement(task maintenance.Task) {
	eh.lock.Lock()
	defer eh.lock.Unlock()
	eh.tasks = append(eh.tasks, maintenance.Task{AggregateID: task.AggregateID, Version: task.Version})
}

func (eh *recordHandler) Handle(ctx context.Context, iter maintenance.Iter) error {
	var task maintenance.Task

	for iter.Next(ctx, &task) {
		eh.SetElement(task)
	}
	return nil
}

type hEvent struct {
	VersionI   uint64 `json:"version"`
	EventTypeI string `json:"eventtype"`
	Data       []byte `json:"data"`
}

type eventHandler struct {
	backupPath string
}

func newEventHandler(backupPath string) *eventHandler {
	return &eventHandler{backupPath: backupPath}
}

func unmarshalPlain(data []byte, v interface{}) error {
	if a, ok := v.(*[]byte); ok {
		*a = data
		return nil
	}
	return fmt.Errorf("unsupported type for unmarshaler %T", v)
}

func handleBackupFile(file **os.File, aggregateID, backupPath string) error {
	if *file != nil {
		if err := (*file).Sync(); err != nil {
			return err
		}
		if err := (*file).Close(); err != nil {
			return nil
		}
	}

	ext := path.Ext(backupPath)
	var err error
	*file, err = os.Create(strings.TrimSuffix(backupPath, ext) + "_" + aggregateID + "_" + time.Now().Format("2006-01-02T15:04:05.000") + ext)
	if err != nil {
		return err
	}

	return nil
}

func backup(file *os.File, eu event.EventUnmarshaler) error {
	var e []byte
	err := eu.Unmarshal(&e)
	if err != nil {
		return err
	}
	event := hEvent{VersionI: eu.Version, EventTypeI: eu.EventType, Data: e}

	b, _ := json.MarshalIndent(event, "", "  ")
	text := fmt.Sprintf(string(b) + "\n")
	if _, err = file.WriteString(text); err != nil {
		return err
	}

	return nil
}

func (eh *eventHandler) Handle(ctx context.Context, iter event.Iter) error {
	var eu event.EventUnmarshaler

	aggregateID := ""
	var file *os.File

	for iter.Next(ctx, &eu) {
		if eu.EventType == "" {
			return errors.New("cannot determine type of event")
		}

		if aggregateID != eu.AggregateId {
			aggregateID = eu.AggregateId

			if err := handleBackupFile(&file, aggregateID, eh.backupPath); err != nil {
				return err
			}
		}

		if err := backup(file, eu); err != nil {
			return err
		}
	}

	if file != nil {
		if err := file.Close(); err != nil {
			return nil
		}
	}

	return nil
}

// PerformMaintenance performs the backup & maintenance of the database
func PerformMaintenance() error {
	ctx := context.Background()

	var config Config
	parser := flags.NewParser(&config, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}
	log.Info(config.String())

	eventStore, err := mongodb.NewEventStore(config.Mongo, nil, mongodb.WithUnmarshaler(unmarshalPlain))
	if err != nil {
		return err
	}

	if err = performMaintenanceWithEventStore(ctx, config, eventStore); err != nil {
		return err
	}

	return nil
}

func performMaintenanceWithEventStore(ctx context.Context, config Config, eventStore *mongodb.EventStore) error {
	handler := newRecordHandler()
	if err := eventStore.Query(ctx, config.NumAggregates, handler); err != nil {
		return err
	}
	versionQueries := []eventstore.VersionQuery{}
	for _, task := range handler.tasks {
		versionQueries = append(versionQueries, eventstore.VersionQuery{AggregateId: task.AggregateID, Version: task.Version})
	}

	log.Info("backing up the events")
	eventHandler := newEventHandler(config.BackupPath)
	if err := eventStore.LoadUpToVersion(ctx, versionQueries, eventHandler); err != nil {
		return err
	}

	log.Info("deleting events...")
	if err := eventStore.RemoveUpToVersion(ctx, versionQueries); err != nil {
		return err
	}

	return nil
}
