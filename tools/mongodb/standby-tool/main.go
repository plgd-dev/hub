package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/internal/math"
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

func (c *StandbyConfig) Validate() error {
	if len(c.Members) == 0 {
		return errors.New("members - no standby members found")
	}
	if c.Delays < 0 {
		return errors.New("delays - must be greater than or equal to 0")
	}
	return nil
}

type SecondaryConfig struct {
	Votes    int `yaml:"votes"`
	Priority int `yaml:"priority"`
}

func (c *SecondaryConfig) Validate() error {
	if c.Votes < 0 {
		return errors.New("votes - must be greater than or equal to 0")
	}
	if c.Priority < 0 {
		return errors.New("priority - must be greater than or equal to 0")
	}
	return nil
}

type ReplicaSetConfig struct {
	ForceUpdate      bool            `yaml:"forceUpdate"`
	MaxWaitsForReady int             `yaml:"maxWaitsForReady"`
	Standby          StandbyConfig   `yaml:"standby"`
	Secondary        SecondaryConfig `yaml:"secondary"`
}

func (c *ReplicaSetConfig) Validate() error {
	if err := c.Standby.Validate(); err != nil {
		return fmt.Errorf("standby.%w", err)
	}
	if err := c.Secondary.Validate(); err != nil {
		return fmt.Errorf("secondary.%w", err)
	}
	if c.MaxWaitsForReady < 1 {
		return errors.New("maxWaitsForReady - must be greater than 1")
	}
	return nil
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
				Timeout time.Duration `yaml:"timeout" json:"timeout"`
				TLS     TLSConfig     `yaml:"tls"`
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
		return fmt.Errorf("invalid .mode value(%v), must be either 'standby' or 'active'", c.Mode)
	}
	if err := c.ReplicaSet.Validate(); err != nil {
		return fmt.Errorf("replicaSet.%w", err)
	}
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log: %w", err)
	}
	if c.Clients.Storage.MongoDB.Timeout <= 0 {
		c.Clients.Storage.MongoDB.Timeout = time.Second * 20
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

const (
	errGetReplicaStatusFmt = "failed to get replica set status: %w"
	errGetReplicaConfigFmt = "failed to get replica set config: %w"
	errConnectToMongoDBFmt = "failed to connect to MongoDB: %w"
	errSetHiddenMembersFmt = "failed to set hidden members: %w"
)

var errConfigNotFound = errors.New("config not found in replica set config")

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
	ctx := context.Background()

	app := &App{
		Config:     cfg,
		certClient: certClient,
		logger:     logger,
	}

	switch app.Config.Mode {
	case "active":
		err = app.setActive(ctx)
	case "standby":
		err = app.setStandby(ctx)
	default:
		logger.Fatalf("Invalid mode %s, must be active or standby", app.Config.Mode)
	}
	if err != nil {
		logger.Fatalf("Failed to set mode: %v", err)
	}

	logger.Info("Done")
}

func (app *App) needForceUpdate(ctx context.Context, member string, status primitive.M) bool {
	members := status["members"].(primitive.A)
	for _, m := range members {
		memberMap := m.(primitive.M)
		if memberMap["name"] == member {
			if memberMap["health"] != float64(1) {
				return true
			}
			c, err := app.connectMongo(ctx, member)
			if err != nil {
				return true
			}
			_ = c.Disconnect(ctx)
		}
	}
	return false
}

func (app *App) setActive(ctx context.Context) error {
	client, primaryMember, err := app.connectToPrimaryMember(ctx, true)
	if err != nil {
		return fmt.Errorf(errConnectToMongoDBFmt, err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	err = app.waitForMembers(ctx, client, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to wait for standby members: %w", err)
	}
	force := app.Config.ReplicaSet.ForceUpdate
	if primaryMember == "" {
		force = true
	}
	secondaryMembers, err := app.getSecondaryMembers(ctx, client, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to get secondary members: %w", err)
	}
	err = app.setSecondaryMembers(ctx, client, force, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to set secondary members: %w", err)
	}
	err = app.setHiddenMembers(ctx, client, force, primaryMember, secondaryMembers)
	if err != nil {
		return fmt.Errorf(errSetHiddenMembersFmt, err)
	}
	if force {
		return nil
	}
	newPrimaryMember, err := app.movePrimary(ctx, client, secondaryMembers)
	if err != nil {
		return fmt.Errorf("failed to move primary: %w", err)
	}
	if newPrimaryMember != primaryMember {
		app.logger.Infof("Setting old primary member %s as hidden", primaryMember)
		client, err = app.connectMongo(ctx, newPrimaryMember)
		if err != nil {
			return fmt.Errorf(errConnectToMongoDBFmt, err)
		}
		defer func() { _ = client.Disconnect(context.Background()) }()
		err = app.setHiddenMembers(ctx, client, false, newPrimaryMember, []string{primaryMember})
		if err != nil {
			return fmt.Errorf(errSetHiddenMembersFmt, err)
		}
	}
	return nil
}

func (app *App) connectToPrimaryMember(ctx context.Context, justTry bool) (*mongo.Client, string, error) {
	host := app.Config.ReplicaSet.Standby.Members[0]
	client, err := app.connectMongo(ctx, host)
	if err != nil {
		return nil, "", fmt.Errorf(errConnectToMongoDBFmt, err)
	}
	closeClient := func() {
		_ = client.Disconnect(context.Background())
	}
	primaryMember, err := app.getPrimaryMember(ctx, client)
	if err != nil {
		closeClient()
		return nil, "", fmt.Errorf("failed to get primary member: %w", err)
	}
	if primaryMember == "" {
		if justTry {
			return client, "", nil
		}
		closeClient()
		return nil, "", errors.New("primary member not found")
	}
	if primaryMember != host {
		primaryClient, err2 := app.connectMongo(ctx, primaryMember)
		if err2 != nil {
			if justTry {
				return client, "", nil
			}
			closeClient()
			return nil, "", fmt.Errorf(errConnectToMongoDBFmt, err2)
		}
		closeClient()
		client = primaryClient
	}
	app.logger.Infof("Primary member: %s", primaryMember)
	return client, primaryMember, nil
}

func (app *App) setStandby(ctx context.Context) error {
	client, primaryMember, err := app.connectToPrimaryMember(ctx, false)
	if err != nil {
		return fmt.Errorf(errConnectToMongoDBFmt, err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	err = app.waitForMembers(ctx, client, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to wait for standby members: %w", err)
	}
	secondaryMembers, err := app.getSecondaryMembers(ctx, client, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to get secondary members: %w", err)
	}
	err = app.setSecondaryMembers(ctx, client, app.Config.ReplicaSet.ForceUpdate, secondaryMembers)
	if err != nil {
		return fmt.Errorf("failed to set secondary members: %w", err)
	}
	err = app.waitForMembers(ctx, client, secondaryMembers)
	if err != nil {
		return fmt.Errorf("failed to wait for secondary members: %w", err)
	}
	err = app.setHiddenMembers(ctx, client, app.Config.ReplicaSet.ForceUpdate, primaryMember, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf(errSetHiddenMembersFmt, err)
	}
	newPrimaryMember, err := app.movePrimary(ctx, client, app.Config.ReplicaSet.Standby.Members)
	if err != nil {
		return fmt.Errorf("failed to move primary: %w", err)
	}
	if newPrimaryMember != primaryMember {
		app.logger.Infof("Setting old primary member %s to hidden", primaryMember)
		newPrimaryMemberClient, err2 := app.connectMongo(ctx, newPrimaryMember)
		if err2 != nil {
			return fmt.Errorf(errConnectToMongoDBFmt, err2)
		}
		defer func() { _ = newPrimaryMemberClient.Disconnect(context.Background()) }()
		err = app.setHiddenMembers(ctx, newPrimaryMemberClient, app.Config.ReplicaSet.ForceUpdate, newPrimaryMember, []string{primaryMember})
		if err != nil {
			return fmt.Errorf(errSetHiddenMembersFmt, err)
		}
	}
	return nil
}

func (app *App) connectMongo(ctx context.Context, host string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI("mongodb://" + host)
	if app.certClient != nil {
		tlsConfig := options.Client().SetTLSConfig(app.certClient.GetTLSConfig())
		clientOptions = clientOptions.SetTLSConfig(tlsConfig.TLSConfig)
	}
	clientOptions = clientOptions.SetDirect(true)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf(errConnectToMongoDBFmt, err)
	}
	ctx2, cancel := context.WithTimeout(ctx, app.Config.Clients.Storage.MongoDB.Timeout)
	defer cancel()
	err = client.Ping(ctx2, nil)
	if err != nil {
		_ = client.Disconnect(ctx2)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

func (app *App) getPrimaryMember(ctx context.Context, client *mongo.Client) (string, error) {
	status, err := app.getStatus(ctx, client)
	if err != nil {
		return "", fmt.Errorf(errGetReplicaStatusFmt, err)
	}
	for _, member := range status["members"].(primitive.A) {
		memberMap := member.(primitive.M)
		if memberMap["state"] == int32(1) && memberMap["health"] == float64(1) {
			return memberMap["name"].(string), nil
		}
	}
	return "", nil
}

func (app *App) waitForMembers(ctx context.Context, client *mongo.Client, members []string) error {
	tried := 0
	app.logger.Infof("Checking if members %v are ready", members)
	for {
		config, err := app.getStatus(ctx, client)
		if err != nil {
			return fmt.Errorf(errGetReplicaConfigFmt, err)
		}
		membersExist := true
		for _, member := range members {
			if !app.memberIsReady(member, config) {
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
		tried++
		if tried > app.Config.ReplicaSet.MaxWaitsForReady {
			return fmt.Errorf("failed to wait for members: retries exceeded (%d)", app.Config.ReplicaSet.MaxWaitsForReady)
		}
		log.Debugf("Members not ready, retrying %v", tried)
	}
	return nil
}

func (app *App) memberIsReady(member string, config primitive.M) bool {
	membersI, ok := getValue(config, "members")
	if !ok {
		return false
	}
	members := membersI.(primitive.A)
	for _, m := range members {
		memberMap := m.(primitive.M)
		if memberMap["name"] == member {
			return memberMap["health"] == float64(1)
		}
	}
	return false
}

func (app *App) getSecondaryMembers(ctx context.Context, client *mongo.Client, standbyMembers []string) ([]string, error) {
	config, err := app.getConfig(ctx, client)
	if err != nil {
		return nil, fmt.Errorf(errGetReplicaConfigFmt, err)
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

func (app *App) setSecondaryMembers(ctx context.Context, client *mongo.Client, force bool, secondaryMembers []string) error {
	app.logger.Infof("Setting secondary members %v", secondaryMembers)
	for _, member := range secondaryMembers {
		config, err := app.getConfig(ctx, client)
		if err != nil {
			return fmt.Errorf(errGetReplicaConfigFmt, err)
		}
		if app.isSecondaryMemberConfigured(member, config) {
			app.logger.Infof("Secondary member %s is correctly configured", member)
			continue
		}
		app.logger.Infof("Configuring secondary member %s", member)
		newConfig, err := app.updateSecondaryMemberConfig(member, config)
		if err != nil {
			return fmt.Errorf("failed to update secondary member %s: %w", member, err)
		}
		if !force {
			status, err := app.getStatus(ctx, client)
			if err != nil {
				return fmt.Errorf(errGetReplicaStatusFmt, err)
			}
			force = app.needForceUpdate(ctx, member, status)
		}
		if err := app.reconfigureRS(ctx, client, newConfig, force); err != nil {
			return fmt.Errorf("failed to configure secondary member %s: %w", member, err)
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
			memberMap["secondaryDelaySecs"].(int64) == int64(0) {
			return true
		}
	}
	return false
}

func (app *App) updateSecondaryMemberConfig(member string, config primitive.M) (primitive.M, error) {
	mongoConfigI, ok := getValue(config, "config")
	if !ok {
		return nil, errConfigNotFound
	}
	mongoConfig := mongoConfigI.(primitive.M)
	newMembers := make(primitive.A, 0)
	members := mongoConfig["members"].(primitive.A)
	for _, m := range members {
		memberMap := m.(primitive.M)
		if memberMap["host"] == member {
			memberMap["hidden"] = false
			memberMap["priority"] = float64(app.Config.ReplicaSet.Secondary.Priority)
			memberMap["votes"] = math.CastTo[int32](app.Config.ReplicaSet.Secondary.Votes)
			memberMap["secondaryDelaySecs"] = int32(0)
		}
		newMembers = append(newMembers, memberMap)
	}
	mongoConfig["members"] = newMembers
	config["config"] = mongoConfig
	return config, nil
}

func (app *App) setHiddenMembers(ctx context.Context, client *mongo.Client, force bool, primaryMember string, standbyMembers []string) error {
	app.logger.Infof("Setting hidden members %v", standbyMembers)
	status, err := app.getStatus(ctx, client)
	if err != nil {
		return fmt.Errorf(errGetReplicaStatusFmt, err)
	}
	for _, member := range standbyMembers {
		config, err := app.getConfig(ctx, client)
		if err != nil {
			return fmt.Errorf(errGetReplicaConfigFmt, err)
		}
		if app.isHiddenMemberConfigured(member, config) {
			app.logger.Infof("Hidden member %s is correctly configured", member)
			continue
		}
		if member == primaryMember {
			app.logger.Infof("Primary member %s cannot be set as hidden, skipping", member)
			continue
		}
		app.logger.Infof("Configuring hidden member %s", member)
		newConfig, err := app.updateHiddenMemberConfig(member, config)
		if err != nil {
			return fmt.Errorf("failed to update hidden member %s: %w", member, err)
		}
		if !force {
			force = app.needForceUpdate(ctx, member, status)
		}
		if err := app.reconfigureRS(ctx, client, newConfig, force); err != nil {
			return fmt.Errorf("failed to configure hidden member %s: %w", member, err)
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
		return nil, errConfigNotFound
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
			memberMap["secondaryDelaySecs"] = int32(app.Config.ReplicaSet.Standby.Delays.Seconds())
		}
		newMembers = append(newMembers, memberMap)
	}
	mongoConfig["members"] = newMembers
	config["config"] = mongoConfig
	return config, nil
}

func (app *App) movePrimary(ctx context.Context, client *mongo.Client, standbyMembers []string) (string, error) {
	for tries := 0; tries < app.Config.ReplicaSet.MaxWaitsForReady; tries++ {
		status, err := app.getStatus(ctx, client)
		if err != nil {
			return "", fmt.Errorf(errGetReplicaStatusFmt, err)
		}
		primaryMember := app.getPrimaryMemberFromStatus(status)
		if primaryMember == "" {
			time.Sleep(1 * time.Second)
			log.Debug("Primary member not found, retrying")
			continue
		}
		if !app.isStandbyMember(primaryMember, standbyMembers) {
			return primaryMember, nil
		}
		// Decrease priority of current primary
		if err2 := app.decreasePrimaryPriority(ctx, client, primaryMember); err2 != nil {
			return "", fmt.Errorf("failed to decrease priority of primary member %s: %w", primaryMember, err2)
		}
		// Initiate step down
		if err2 := app.stepDownPrimary(ctx, client, primaryMember); err2 != nil {
			return "", fmt.Errorf("failed to step down primary member %s: %w", primaryMember, err2)
		}
		if tries > 0 {
			time.Sleep(1 * time.Second)
			log.Debugf("Primary member not found, retrying %v", tries)
		}
	}
	return "", fmt.Errorf("primary member not found after %v attempts", app.Config.ReplicaSet.MaxWaitsForReady)
}

func (app *App) decreasePrimaryPriority(ctx context.Context, client *mongo.Client, primaryMember string) error {
	config, err := app.getConfig(ctx, client)
	if err != nil {
		return fmt.Errorf(errGetReplicaConfigFmt, err)
	}
	newConfig, err := app.updatePrimaryPriority(primaryMember, config)
	if err != nil {
		return fmt.Errorf("failed to update primary member priority: %w", err)
	}
	status, err := app.getStatus(ctx, client)
	if err != nil {
		return fmt.Errorf(errGetReplicaStatusFmt, err)
	}
	return app.reconfigureRS(ctx, client, newConfig, app.needForceUpdate(ctx, primaryMember, status))
}

func (app *App) updatePrimaryPriority(primaryMember string, config primitive.M) (primitive.M, error) {
	mongoConfigI, ok := getValue(config, "config")
	if !ok {
		return nil, errConfigNotFound
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

func (app *App) stepDownPrimary(ctx context.Context, client *mongo.Client, primaryMember string) error {
	command := bson.D{
		{Key: "replSetStepDown", Value: int(app.Config.Clients.Storage.MongoDB.Timeout.Seconds())},
	}
	var result primitive.M
	ctx2, cancel := context.WithTimeout(ctx, app.Config.Clients.Storage.MongoDB.Timeout)
	defer cancel()
	err := client.Database("admin").RunCommand(ctx2, command).Decode(&result)
	if err != nil {
		return fmt.Errorf("failed to step down primary member %s: %w", primaryMember, err)
	}
	return nil
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
	return getValue(sub, keys[1:]...) //nolint:gosec
}

func (app *App) getStatus(ctx context.Context, client *mongo.Client) (primitive.M, error) {
	var result primitive.M
	ctx2, cancel := context.WithTimeout(ctx, app.Config.Clients.Storage.MongoDB.Timeout)
	defer cancel()
	err := client.Database("admin").RunCommand(ctx2, bson.D{{Key: "replSetGetStatus", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (app *App) getConfig(ctx context.Context, client *mongo.Client) (primitive.M, error) {
	var result primitive.M
	ctx2, cancel := context.WithTimeout(ctx, app.Config.Clients.Storage.MongoDB.Timeout)
	defer cancel()
	err := client.Database("admin").RunCommand(ctx2, bson.D{{Key: "replSetGetConfig", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (app *App) reconfigureRS(ctx context.Context, client *mongo.Client, newConfig primitive.M, force bool) error {
	configI := newConfig["config"]
	config := configI.(primitive.M)
	config["version"] = config["version"].(int32) + 1

	app.logger.Infof("Reconfiguring replica set with version(%v) and force flag(%v)", config["version"], force)

	command := bson.D{
		{Key: "replSetReconfig", Value: config},
		{Key: "force", Value: force},
	}
	var result primitive.M
	ctx2, cancel := context.WithTimeout(ctx, app.Config.Clients.Storage.MongoDB.Timeout)
	defer cancel()
	err := client.Database("admin").RunCommand(ctx2, command).Decode(&result)
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
