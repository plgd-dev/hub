package store

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
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

func errInvalidArgument(err error) error {
	return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
}

func ValidateAndNormalize(c *pb.Configuration) error {
	if _, err := uuid.Parse(c.GetId()); err != nil {
		return errInvalidArgument(fmt.Errorf("invalid configuration ID(%v): %w", c.GetId(), err))
	}
	if len(c.GetResources()) == 0 {
		return errInvalidArgument(errors.New("invalid configuration resources"))
	}
	if c.GetOwner() == "" {
		return errInvalidArgument(errors.New("empty configuration owner"))
	}
	slices.SortFunc(c.GetResources(), func(i, j *pb.Configuration_Resource) int {
		return strings.Compare(i.GetHref(), j.GetHref())
	})
	return nil
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
