package store

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
)

func ValidateAndNormalizeConfiguration(c *pb.Configuration, isUpdate bool) error {
	if isUpdate || c.GetId() != "" {
		if _, err := uuid.Parse(c.GetId()); err != nil {
			return errInvalidArgument(fmt.Errorf("invalid ID(%v): %w", c.GetId(), err))
		}
	}
	if c.GetOwner() == "" {
		return errInvalidArgument(errors.New("missing owner"))
	}
	if len(c.GetResources()) == 0 {
		return errInvalidArgument(errors.New("missing resources"))
	}
	resources := slices.Clone(c.GetResources())
	slices.SortFunc(resources, func(i, j *pb.Configuration_Resource) int {
		return strings.Compare(i.GetHref(), j.GetHref())
	})
	resources = slices.CompactFunc(resources, func(i, j *pb.Configuration_Resource) bool {
		return i.GetHref() == j.GetHref()
	})
	c.Resources = resources
	return nil
}

type ConfigurationVersion struct {
	Version   uint64                       `bson:"version"`
	Resources []*pb.Configuration_Resource `bson:"resources"`
}

type Configuration struct {
	Id        string                 `bson:"_id"`
	Name      string                 `bson:"name,omitempty"`
	Owner     string                 `bson:"owner"`
	Timestamp int64                  `bson:"timestamp"`
	Versions  []ConfigurationVersion `bson:"versions,omitempty"`
}

func MakeConfiguration(c *pb.Configuration) Configuration {
	return Configuration{
		Id:        c.GetId(),
		Name:      c.GetName(),
		Owner:     c.GetOwner(),
		Timestamp: GetTimestampOrNow(c.GetTimestamp()),
		Versions:  []ConfigurationVersion{{Version: c.GetVersion(), Resources: c.GetResources()}},
	}
}

func (c *Configuration) Clone() *Configuration {
	return &Configuration{
		Id:        c.Id,
		Name:      c.Name,
		Owner:     c.Owner,
		Timestamp: c.Timestamp,
		Versions:  slices.Clone(c.Versions),
	}
}

func (c *Configuration) GetConfiguration(version int) *pb.Configuration {
	return &pb.Configuration{
		Id:        c.Id,
		Name:      c.Name,
		Owner:     c.Owner,
		Timestamp: c.Timestamp,
		Version:   c.Versions[version].Version,
		Resources: c.Versions[version].Resources,
	}
}
