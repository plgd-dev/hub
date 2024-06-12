package store

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
)

func checkConfigurationId(c string, isUpdate bool) error {
	if isUpdate && c == "" {
		// in this case the update will keep the configuration ID already in the database
		return nil
	}
	if _, err := uuid.Parse(c); err != nil {
		return errInvalidArgument(fmt.Errorf("invalid configuration ID(%v): %w", c, err))
	}
	return nil
}

func ValidateAndNormalizeCondition(c *pb.Condition, isUpdate bool) error {
	if isUpdate || c.GetId() != "" {
		if _, err := uuid.Parse(c.GetId()); err != nil {
			return errInvalidArgument(fmt.Errorf("invalid ID(%v): %w", c.GetId(), err))
		}
	}
	if err := checkConfigurationId(c.GetConfigurationId(), isUpdate); err != nil {
		return errInvalidArgument(fmt.Errorf("invalid configuration ID(%v): %w", c.GetConfigurationId(), err))
	}
	if c.GetOwner() == "" {
		return errInvalidArgument(errors.New("missing owner"))
	}
	// ensure that filter arrays are sorted and compacted, so we can query for exact match instead of other more expensive queries
	c.DeviceIdFilter = strings.Unique(c.GetDeviceIdFilter())
	c.ResourceTypeFilter = strings.Unique(c.GetResourceTypeFilter())
	c.ResourceHrefFilter = strings.Unique(c.GetResourceHrefFilter())
	return nil
}

type ConditionVersion struct {
	Name               string   `bson:"name,omitempty"`
	Version            uint64   `bson:"version"`
	Enabled            bool     `bson:"enabled"`
	Timestamp          int64    `bson:"timestamp"`
	DeviceIdFilter     []string `bson:"deviceIdFilter,omitempty"`
	ResourceTypeFilter []string `bson:"resourceTypeFilter,omitempty"`
	ResourceHrefFilter []string `bson:"resourceHrefFilter,omitempty"`
	JqExpressionFilter string   `bson:"jqExpressionFilter,omitempty"`
	ApiAccessToken     string   `bson:"apiAccessToken,omitempty"`
}

func (cv *ConditionVersion) Copy() ConditionVersion {
	return ConditionVersion{
		Name:               cv.Name,
		Version:            cv.Version,
		Enabled:            cv.Enabled,
		Timestamp:          cv.Timestamp,
		DeviceIdFilter:     slices.Clone(cv.DeviceIdFilter),
		ResourceTypeFilter: slices.Clone(cv.ResourceTypeFilter),
		ResourceHrefFilter: slices.Clone(cv.ResourceHrefFilter),
		JqExpressionFilter: cv.JqExpressionFilter,
		ApiAccessToken:     cv.ApiAccessToken,
	}
}

func MakeConditionVersion(c *pb.Condition) ConditionVersion {
	return ConditionVersion{
		Name:               c.GetName(),
		Version:            c.GetVersion(),
		Enabled:            c.GetEnabled(),
		Timestamp:          c.GetTimestamp(),
		DeviceIdFilter:     c.GetDeviceIdFilter(),
		ResourceTypeFilter: c.GetResourceTypeFilter(),
		ResourceHrefFilter: c.GetResourceHrefFilter(),
		JqExpressionFilter: c.GetJqExpressionFilter(),
		ApiAccessToken:     c.GetApiAccessToken(),
	}
}

type Condition struct {
	Id              string             `bson:"_id"`
	Owner           string             `bson:"owner"`
	ConfigurationId string             `bson:"configurationId"`
	Latest          *ConditionVersion  `bson:"latest,omitempty"`
	Versions        []ConditionVersion `bson:"versions,omitempty"`
}

func MakeFirstCondition(c *pb.Condition) Condition {
	version := MakeConditionVersion(c)
	return Condition{
		Id:              c.GetId(),
		Owner:           c.GetOwner(),
		ConfigurationId: c.GetConfigurationId(),
		Latest:          &version,
		Versions:        []ConditionVersion{version},
	}
}

func (c *Condition) GetLatest() (*pb.Condition, error) {
	if c.Latest == nil {
		return nil, errors.New("latest condition not set")
	}
	return &pb.Condition{
		Id:                 c.Id,
		Owner:              c.Owner,
		ConfigurationId:    c.ConfigurationId,
		Name:               c.Latest.Name,
		Enabled:            c.Latest.Enabled,
		Version:            c.Latest.Version,
		Timestamp:          c.Latest.Timestamp,
		DeviceIdFilter:     c.Latest.DeviceIdFilter,
		ResourceTypeFilter: c.Latest.ResourceTypeFilter,
		ResourceHrefFilter: c.Latest.ResourceHrefFilter,
		JqExpressionFilter: c.Latest.JqExpressionFilter,
		ApiAccessToken:     c.Latest.ApiAccessToken,
	}, nil
}

func (c *Condition) GetCondition(index int) *pb.Condition {
	return &pb.Condition{
		Id:                 c.Id,
		Owner:              c.Owner,
		ConfigurationId:    c.ConfigurationId,
		Name:               c.Versions[index].Name,
		Enabled:            c.Versions[index].Enabled,
		Version:            c.Versions[index].Version,
		Timestamp:          c.Versions[index].Timestamp,
		DeviceIdFilter:     c.Versions[index].DeviceIdFilter,
		ResourceTypeFilter: c.Versions[index].ResourceTypeFilter,
		ResourceHrefFilter: c.Versions[index].ResourceHrefFilter,
		JqExpressionFilter: c.Versions[index].JqExpressionFilter,
		ApiAccessToken:     c.Versions[index].ApiAccessToken,
	}
}

func (c *Condition) Clone() *Condition {
	c2 := &Condition{
		Id:              c.Id,
		Owner:           c.Owner,
		ConfigurationId: c.ConfigurationId,
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

func ValidateAndNormalizeConditionsQuery(q *GetLatestConditionsQuery) error {
	if q.DeviceID == "" && q.ResourceHref == "" && len(q.ResourceTypeFilter) == 0 {
		return errInvalidArgument(errors.New("at least one condition filter must be set"))
	}
	q.ResourceTypeFilter = strings.Unique(q.ResourceTypeFilter)
	return nil
}
