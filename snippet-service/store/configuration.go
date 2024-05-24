package store

import (
	"slices"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
)

const (
	VersionsKey = "versions" // must match with Versions field tag
)

type ConfigurationVersion struct {
	Version   uint64                       `bson:"version"`
	Resources []*pb.Configuration_Resource `bson:"resources"`
}

type Configuration struct {
	Id       string                 `bson:"_id"`
	Name     string                 `bson:"name,omitempty"`
	Owner    string                 `bson:"owner"`
	Versions []ConfigurationVersion `bson:"versions,omitempty"`
}

func MakeConfiguration(c *pb.Configuration) Configuration {
	return Configuration{
		Id:       c.GetId(),
		Name:     c.GetName(),
		Owner:    c.GetOwner(),
		Versions: []ConfigurationVersion{{Version: c.GetVersion(), Resources: c.GetResources()}},
	}
}

func (c *Configuration) Clone() *Configuration {
	return &Configuration{
		Id:       c.Id,
		Name:     c.Name,
		Owner:    c.Owner,
		Versions: slices.Clone(c.Versions),
	}
}
