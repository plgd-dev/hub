package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/build"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StandbyConfig struct {
	Members []string      `yaml:"members"`
	Delays  time.Duration `yaml:"delays"`
}

type SecondaryConfig struct {
	Delays   time.Duration `yaml:"delays"`
	Votes    int           `yaml:"votes"`
	Priority int           `yaml:"priority"`
}

type ReplicaSetConfig struct {
	Standby   StandbyConfig   `yaml:"standby"`
	Secondary SecondaryConfig `yaml:"secondary"`
}

type TLSConfig struct {
	Enabled bool          `yaml:"enabled"`
	TLS     client.Config `yaml:",inline"`
}

type Config struct {
	Mode       string           `yaml:"mode"`
	ReplicaSet ReplicaSetConfig `yaml:"replicaSet"`
	Clients    struct {
		Storage struct {
			MongoDB struct {
				TLS TLSConfig `yaml:"tls"`
			} `yaml:"mongoDB"`
		} `yaml:"storage"`
	} `yaml:"clients"`
	Log log.Config `yaml:"log"`
}

func (c Config) String() string {
	return config.ToString(c)
}

func (c *Config) Validate() error {
	if c.Mode != "standby" && c.Mode != "active" {
		return errors.New("invalid .mode value, must be either 'standby' or 'active'")
	}
	if len(c.ReplicaSet.Standby.Members) == 0 {
		return errors.New("no standby members found")
	}
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log: %w", err)
	}
	if c.Clients.Storage.MongoDB.TLS.Enabled {
		if err := c.Clients.Storage.MongoDB.TLS.TLS.Validate(); err != nil {
			return fmt.Errorf("clients.storage.mongoDB.tls: %w", err)
		}
	}
	return nil
}

type App struct {
	Config     Config
	logger     log.Logger
	certClient *client.CertManager
}

func main() {
	var cfg Config
	err := config.LoadAndValidateConfig(&cfg)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}
	logger := log.NewLogger(cfg.Log)
	log.Set(logger)
	logger.Debugf("version: %v, buildDate: %v, buildRevision %v", build.Version, build.BuildDate, build.CommitHash)
	logger.Infof("config: %v", cfg.String())

	fileWatcher, err := fsnotify.NewWatcher(logger)
	if err != nil {
		logger.Fatalf("cannot create file fileWatcher: %v", err)
	}
	var certClient *client.CertManager
	if cfg.Clients.Storage.MongoDB.TLS.Enabled {
		certClient, err = client.New(cfg.Clients.Storage.MongoDB.TLS.TLS, fileWatcher, logger)
		if err != nil {
			logger.Fatalf("cannot create cert client: %v", err)
		}
		defer func() {
			certClient.Close()
			_ = fileWatcher.Close()
		}()
	} else {
		defer func() {
			_ = fileWatcher.Close()
		}()
	}

	app := &App{
		Config:     cfg,
		certClient: certClient,
		logger:     logger,
	}

	switch app.Config.Mode {
	case "active":
		err = app.setActive()
	case "standby":
		err = app.setStandby()
	default:
		logger.Fatalf("Invalid mode %s, must be active or standby", app.Config.Mode)
	}
	if err != nil {
		logger.Fatalf("Failed to set mode: %v", err)
	}

	logger.Info("Done")
}

func (app *App) setActive() error {
	host := app.Config.ReplicaSet.Standby.Members[0]
	client, err := app.connectMongo(host)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()
	primaryMember, err := app.getPrimaryMember(client)
	if err != nil {
		return fmt.Errorf("failed to get primary member: %w", err)
	}
	if primaryMember != "" {
		app.logger.Infof("Primary member is %s", primaryMember)
		host = primaryMember
		client, err = app.connectMongo(host)
		if err != nil {
			return fmt.Errorf("failed to connect to MongoDB: %w", err)
		}
		defer func() { _ = client.Disconnect(context.Background()) }()
	}

	err = app.waitForStandbyMembers(client)
	if err != nil {
		return fmt.Errorf("failed to wait for standby members: %w", err)
	}
	force := true
	if primaryMember != "" {
		force = false
	}
	secondaryMembers, err := app.getSecondaryMembers(client, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to get secondary members: %w", err)
	}
	err = app.setSecondaryMembers(client, force, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to set secondary members: %w", err)
	}

	err = app.setHiddenMembers(client, force, primaryMember, secondaryMembers)
	if err != nil {
		return fmt.Errorf("failed to set hidden members: %w", err)
	}
	newPrimaryMember, err := app.movePrimary(client, secondaryMembers)
	if err != nil {
		return fmt.Errorf("failed to move primary: %w", err)
	}
	if newPrimaryMember != primaryMember {
		app.logger.Infof("Setting old primary member %s as hidden", primaryMember)
		client, err = app.connectMongo(newPrimaryMember)
		if err != nil {
			return fmt.Errorf("failed to connect to MongoDB: %w", err)
		}
		defer func() { _ = client.Disconnect(context.Background()) }()
		err = app.setHiddenMembers(client, false, newPrimaryMember, []string{primaryMember})
		if err != nil {
			return fmt.Errorf("failed to set hidden members: %w", err)
		}
	}
	return nil
}

func (app *App) setStandby() error {
	host := app.Config.ReplicaSet.Standby.Members[0]
	client, err := app.connectMongo(host)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()
	primaryMember, err := app.getPrimaryMember(client)
	if err != nil {
		return fmt.Errorf("failed to get primary member: %w", err)
	}
	if primaryMember == "" {
		return errors.New("primary member not found")
	}
	app.logger.Infof("Primary member: %s", primaryMember)
	err = app.waitForStandbyMembers(client)
	if err != nil {
		return fmt.Errorf("failed to wait for standby members: %w", err)
	}
	secondaryMembers, err := app.getSecondaryMembers(client, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to get secondary members: %w", err)
	}
	err = app.setSecondaryMembers(client, false, secondaryMembers)
	if err != nil {
		return fmt.Errorf("failed to set secondary members: %w", err)
	}
	client, err = app.connectMongo(primaryMember)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()
	err = app.setHiddenMembers(client, false, primaryMember, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to set hidden members: %w", err)
	}
	newPrimaryMember, err := app.movePrimary(client, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to move primary: %w", err)
	}
	if newPrimaryMember != primaryMember {
		app.logger.Infof("Setting old primary member %s to hidden", primaryMember)
		client, err = app.connectMongo(newPrimaryMember)
		if err != nil {
			return fmt.Errorf("failed to connect to MongoDB: %w", err)
		}
		defer func() { _ = client.Disconnect(context.Background()) }()
		err = app.setHiddenMembers(client, false, newPrimaryMember, []string{primaryMember})
		if err != nil {
			return fmt.Errorf("failed to set hidden members: %w", err)
		}
	}
	return nil
}

func (app *App) connectMongo(host string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI("mongodb://" + host)
	if app.certClient != nil {
		tlsConfig := options.Client().SetTLSConfig(app.certClient.GetTLSConfig())
		clientOptions = clientOptions.SetTLSConfig(tlsConfig.TLSConfig)
	}
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	return client, nil
}

func (app *App) getPrimaryMember(client *mongo.Client) (string, error) {
	status, err := app.getStatus(client)
	if err != nil {
		return "", fmt.Errorf("failed to get replica set status: %w", err)
	}
	for _, member := range status["members"].(primitive.A) {
		memberMap := member.(primitive.M)
		if memberMap["stateStr"] == "PRIMARY" {
			return memberMap["name"].(string), nil
		}
	}
	return "", nil
}

func (app *App) waitForStandbyMembers(client *mongo.Client) error {
	for {
		app.logger.Info("Checking if all standby members are ready")
		config, err := app.getConfig(client)
		if err != nil {
			return fmt.Errorf("failed to get replica set config: %w", err)
		}
		standbyMembers := app.Config.ReplicaSet.Standby.Members
		membersExist := true
		for _, member := range standbyMembers {
			if !app.memberExists(member, config) {
				app.logger.Infof("Member %s is not ready", member)
				membersExist = false
				break
			}
			app.logger.Infof("Member %s is ready", member)
		}
		if membersExist {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func (app *App) memberExists(member string, config primitive.M) bool {
	membersI, ok := getValue(config, "config", "members")
	if !ok {
		return false
	}
	members := membersI.(primitive.A)
	for _, m := range members {
		memberMap := m.(primitive.M)
		if memberMap["host"] == member {
			return true
		}
	}
	return false
}

func (app *App) getSecondaryMembers(client *mongo.Client, standbyMembers []string) ([]string, error) {
	config, err := app.getConfig(client)
	if err != nil {
		return nil, fmt.Errorf("failed to get replica set config: %w", err)
	}
	var secondaryMembers []string
	membersI, ok := getValue(config, "config", "members")
	if !ok {
		return nil, errors.New("members not found in replica set config")
	}
	members := membersI.(primitive.A)
	for _, member := range members {
		memberMap := member.(primitive.M)
		host := memberMap["host"].(string)
		if !app.isStandbyMember(host, standbyMembers) {
			secondaryMembers = append(secondaryMembers, host)
		}
	}
	return secondaryMembers, nil
}

func (app *App) isStandbyMember(host string, standbyMembers []string) bool {
	for _, standby := range standbyMembers {
		if host == standby {
			return true
		}
	}
	return false
}

func (app *App) setSecondaryMembers(client *mongo.Client, force bool, secondaryMembers []string) error {
	app.logger.Infof("Setting secondary members %v", secondaryMembers)
	for _, member := range secondaryMembers {
		config, err := app.getConfig(client)
		if err != nil {
			return fmt.Errorf("failed to get replica set config: %w", err)
		}
		if !app.isSecondaryMemberConfigured(member, config) {
			app.logger.Infof("Configuring secondary member %s", member)
			newConfig, err := app.updateSecondaryMemberConfig(member, config)
			if err != nil {
				return fmt.Errorf("failed to update secondary member %s: %w", member, err)
			}
			if err := app.reconfigureRS(client, newConfig, force); err != nil {
				return fmt.Errorf("failed to configure secondary member %s: %w", member, err)
			}
		} else {
			app.logger.Infof("Secondary member %s is correctly configured", member)
		}
	}
	return nil
}

func (app *App) isSecondaryMemberConfigured(member string, config primitive.M) bool {
	memberI, ok := getValue(config, "config", "members")
	if !ok {
		return false
	}
	members := memberI.(primitive.A)
	for _, m := range members {
		memberMap := m.(primitive.M)
		if memberMap["host"] == member &&
			!memberMap["hidden"].(bool) &&
			memberMap["priority"].(float64) > 0 &&
			memberMap["votes"].(int32) > 0 &&
			float64(memberMap["secondaryDelaySecs"].(int64)) == app.Config.ReplicaSet.Secondary.Delays.Seconds() {
			return true
		}
	}
	return false
}

func (app *App) updateSecondaryMemberConfig(member string, config primitive.M) (primitive.M, error) {
	mongoConfigI, ok := getValue(config, "config")
	if !ok {
		return nil, errors.New("config not found in replica set config")
	}
	mongoConfig := mongoConfigI.(primitive.M)
	newMembers := make(primitive.A, 0)
	members := mongoConfig["members"].(primitive.A)
	for _, m := range members {
		memberMap := m.(primitive.M)
		if memberMap["host"] == member {
			memberMap["hidden"] = false
			memberMap["priority"] = float64(app.Config.ReplicaSet.Secondary.Priority)
			memberMap["votes"] = int32(app.Config.ReplicaSet.Secondary.Votes)
			memberMap["secondaryDelaySecs"] = int64(app.Config.ReplicaSet.Secondary.Delays.Seconds())
		}
		newMembers = append(newMembers, memberMap)
	}
	mongoConfig["members"] = newMembers
	config["config"] = mongoConfig
	return config, nil
}

func (app *App) setHiddenMembers(client *mongo.Client, force bool, primaryMember string, standbyMembers []string) error {
	app.logger.Infof("Setting hidden members %v", standbyMembers)
	for _, member := range standbyMembers {
		config, err := app.getConfig(client)
		if err != nil {
			return fmt.Errorf("failed to get replica set config: %w", err)
		}
		if !app.isHiddenMemberConfigured(member, config) {
			if member == primaryMember {
				app.logger.Infof("Primary member %s cannot be set as hidden, skipping", member)
				continue
			}
			app.logger.Infof("Configuring hidden member %s", member)
			newConfig, err := app.updateHiddenMemberConfig(member, config)
			if err != nil {
				return fmt.Errorf("failed to update hidden member %s: %w", member, err)
			}
			if err := app.reconfigureRS(client, newConfig, force); err != nil {
				return fmt.Errorf("failed to configure hidden member %s: %w", member, err)
			}
		} else {
			app.logger.Infof("Hidden member %s is correctly configured", member)
		}
	}
	return nil
}

func (app *App) isHiddenMemberConfigured(member string, config primitive.M) bool {
	membersI, ok := getValue(config, "config", "members")
	if !ok {
		return false
	}
	members := membersI.(primitive.A)
	for _, m := range members {
		memberMap := m.(primitive.M)
		if memberMap["host"] == member &&
			memberMap["hidden"].(bool) &&
			memberMap["priority"].(float64) == 0 &&
			memberMap["votes"].(int32) == 0 &&
			float64(memberMap["secondaryDelaySecs"].(int64)) == app.Config.ReplicaSet.Standby.Delays.Seconds() {
			return true
		}
	}
	return false
}

func (app *App) updateHiddenMemberConfig(member string, config primitive.M) (primitive.M, error) {
	mongoConfigI, ok := getValue(config, "config")
	if !ok {
		return nil, errors.New("config not found in replica set config")
	}
	mongoConfig := mongoConfigI.(primitive.M)
	newMembers := make(primitive.A, 0)
	members := mongoConfig["members"].(primitive.A)
	for _, m := range members {
		memberMap := m.(primitive.M)
		if memberMap["host"] == member {
			memberMap["hidden"] = true
			memberMap["priority"] = int32(0)
			memberMap["votes"] = int32(0)
			memberMap["secondaryDelaySecs"] = int32(app.Config.ReplicaSet.Standby.Delays)
		}
		newMembers = append(newMembers, memberMap)
	}
	mongoConfig["members"] = newMembers
	config["config"] = mongoConfig
	return config, nil
}

func (app *App) movePrimary(client *mongo.Client, standbyMembers []string) (string, error) {
	for numTries := 0; numTries < 60; numTries++ {
		status, err := app.getStatus(client)
		if err != nil {
			return "", fmt.Errorf("failed to get replica set status: %w", err)
		}
		primaryMember := app.getPrimaryMemberFromStatus(status)
		if !app.isStandbyMember(primaryMember, standbyMembers) {
			return primaryMember, nil
		}
		// Decrease priority of current primary
		if err2 := app.decreasePrimaryPriority(client, primaryMember); err2 != nil {
			return "", fmt.Errorf("failed to decrease priority of primary member %s: %w", primaryMember, err2)
		}
		// Initiate step down
		if err2 := app.stepDownPrimary(client, primaryMember); err2 != nil {
			return "", fmt.Errorf("failed to step down primary member %s: %w", primaryMember, err2)
		}
		if numTries > 0 {
			time.Sleep(1 * time.Second)
			log.Info("Primary member not found, retrying")
		}
	}
	return "", errors.New("primary member not found after 60 attempts")
}

func (app *App) decreasePrimaryPriority(client *mongo.Client, primaryMember string) error {
	config, err := app.getConfig(client)
	if err != nil {
		return fmt.Errorf("failed to get replica set config: %w", err)
	}
	newConfig, err := app.updatePrimaryPriority(primaryMember, config)
	if err != nil {
		return fmt.Errorf("failed to update primary member priority: %w", err)
	}
	return app.reconfigureRS(client, newConfig, false)
}

func (app *App) updatePrimaryPriority(primaryMember string, config primitive.M) (primitive.M, error) {
	mongoConfigI, ok := getValue(config, "config")
	if !ok {
		return nil, errors.New("config not found in replica set config")
	}
	mongoConfig := mongoConfigI.(primitive.M)
	newMembers := make(primitive.A, 0)
	members := mongoConfig["members"].(primitive.A)

	for _, m := range members {
		memberMap := m.(primitive.M)
		if memberMap["host"] == primaryMember {
			memberMap["priority"] = float64(0.1)
		}
		newMembers = append(newMembers, memberMap)
	}
	mongoConfig["members"] = newMembers
	config["config"] = mongoConfig
	return config, nil
}

func (app *App) stepDownPrimary(client *mongo.Client, primaryMember string) error {
	command := bson.D{
		{Key: "replSetStepDown", Value: 60}, // 60 seconds timeout
	}
	var result primitive.M
	err := client.Database("admin").RunCommand(context.Background(), command).Decode(&result)
	if err != nil {
		return fmt.Errorf("failed to step down primary member %s: %w", primaryMember, err)
	}
	return nil
}

func (app *App) getStatus(client *mongo.Client) (primitive.M, error) {
	var result primitive.M
	err := client.Database("admin").RunCommand(context.Background(), bson.D{{Key: "replSetGetStatus", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getValue(v primitive.M, keys ...string) (interface{}, bool) {
	if len(keys) == 0 {
		return nil, false
	}
	if len(keys) == 1 {
		value, ok := v[keys[0]]
		return value, ok
	}
	sub, ok := v[keys[0]].(primitive.M)
	if !ok {
		return nil, false
	}
	return getValue(sub, keys[1:]...)
}

func (app *App) getConfig(client *mongo.Client) (primitive.M, error) {
	var result primitive.M
	err := client.Database("admin").RunCommand(context.Background(), bson.D{{Key: "replSetGetConfig", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (app *App) reconfigureRS(client *mongo.Client, newConfig primitive.M, force bool) error {
	configI := newConfig["config"]
	config := configI.(primitive.M)
	config["version"] = config["version"].(int32) + 1

	command := bson.D{
		{Key: "replSetReconfig", Value: config},
		{Key: "force", Value: force},
	}
	var result primitive.M
	err := client.Database("admin").RunCommand(context.Background(), command).Decode(&result)
	if err != nil {
		return fmt.Errorf("failed to reconfigure replica set: %w", err)
	}
	return nil
}

func (app *App) getPrimaryMemberFromStatus(status primitive.M) string {
	members := status["members"].(primitive.A)
	for _, member := range members {
		memberMap := member.(primitive.M)
		if memberMap["state"].(int32) == 1 {
			return memberMap["name"].(string)
		}
	}
	return ""
}
