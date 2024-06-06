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
	Name      string                       `bson:"name,omitempty"`
	Version   uint64                       `bson:"version"`
	Resources []*pb.Configuration_Resource `bson:"resources"`
	Timestamp int64                        `bson:"timestamp"`
}

func (cv *ConfigurationVersion) Copy() ConfigurationVersion {
	c := ConfigurationVersion{
		Name:      cv.Name,
		Version:   cv.Version,
		Timestamp: cv.Timestamp,
	}
	for _, r := range cv.Resources {
		c.Resources = append(c.Resources, r.Clone())
	}
	return c
}

func MakeConfigurationVersion2(c *pb.Configuration) ConfigurationVersion {
	return ConfigurationVersion{
		Name:      c.GetName(),
		Version:   c.GetVersion(),
		Resources: c.GetResources(),
		Timestamp: c.GetTimestamp(),
	}
}

type Configuration struct {
	Id       string                 `bson:"_id"`
	Owner    string                 `bson:"owner"`
	Latest   *ConfigurationVersion  `bson:"latest,omitempty"`
	Versions []ConfigurationVersion `bson:"versions,omitempty"`
}

func MakeFirstConfiguration2(c *pb.Configuration) Configuration {
	version := ConfigurationVersion{
		Name:      c.GetName(),
		Version:   c.GetVersion(),
		Resources: c.GetResources(),
		Timestamp: c.GetTimestamp(),
	}

	return Configuration{
		Id:       c.GetId(),
		Owner:    c.GetOwner(),
		Latest:   &version,
		Versions: []ConfigurationVersion{version},
	}
}

func (c *Configuration) GetLatest() (*pb.Configuration, error) {
	if c.Latest == nil {
		return nil, errors.New("latest configuration not set")
	}
	return &pb.Configuration{
		Id:        c.Id,
		Owner:     c.Owner,
		Version:   c.Latest.Version,
		Name:      c.Latest.Name,
		Resources: c.Latest.Resources,
		Timestamp: c.Latest.Timestamp,
	}, nil
}

func (c *Configuration) GetConfiguration(index int) *pb.Configuration {
	return &pb.Configuration{
		Id:        c.Id,
		Owner:     c.Owner,
		Name:      c.Versions[index].Name,
		Timestamp: c.Versions[index].Timestamp,
		Version:   c.Versions[index].Version,
		Resources: c.Versions[index].Resources,
	}
}

func (c *Configuration) Clone() *Configuration {
	c2 := &Configuration{
		Id:    c.Id,
		Owner: c.Owner,
	}
	if c.Latest != nil {
		latest := c.Latest.Copy()
		c2.Latest = &latest
	}

	for _, v := range c.Versions {
		c2.Versions = append(c2.Versions, v.Copy())
	}
	return c2
}
