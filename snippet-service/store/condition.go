package store

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
)

func normalizeSlice(s []string) []string {
	slices.Sort(s)
	return slices.Compact(s)
}

func GetTimestampOrNow(ts int64) int64 {
	if ts > 0 {
		return ts
	}
	return time.Now().UnixNano()
}

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
	c.DeviceIdFilter = normalizeSlice(c.GetDeviceIdFilter())
	c.ResourceTypeFilter = normalizeSlice(c.GetResourceTypeFilter())
	c.ResourceHrefFilter = normalizeSlice(c.GetResourceHrefFilter())
	return nil
}

type ConditionVersion struct {
	Version            uint64   `bson:"version"`
	DeviceIdFilter     []string `bson:"deviceIdFilter,omitempty"`
	ResourceTypeFilter []string `bson:"resourceTypeFilter,omitempty"`
	ResourceHrefFilter []string `bson:"resourceHrefFilter,omitempty"`
	JqExpressionFilter string   `bson:"jqExpressionFilter,omitempty"`
}

type Condition struct {
	Id              string             `bson:"_id"`
	Name            string             `bson:"name,omitempty"`
	Enabled         bool               `bson:"enabled"`
	Owner           string             `bson:"owner"`
	ConfigurationId string             `bson:"configurationId"`
	ApiAccessToken  string             `bson:"apiAccessToken,omitempty"`
	Timestamp       int64              `bson:"timestamp"`
	Versions        []ConditionVersion `bson:"versions,omitempty"`
}

func MakeCondition(c *pb.Condition) Condition {
	return Condition{
		Id:              c.GetId(),
		Name:            c.GetName(),
		Enabled:         c.GetEnabled(),
		Owner:           c.GetOwner(),
		ConfigurationId: c.GetConfigurationId(),
		ApiAccessToken:  c.GetApiAccessToken(),
		Timestamp:       GetTimestampOrNow(c.GetTimestamp()),
		Versions: []ConditionVersion{
			{
				Version:            c.GetVersion(),
				DeviceIdFilter:     c.GetDeviceIdFilter(),
				ResourceTypeFilter: c.GetResourceTypeFilter(),
				ResourceHrefFilter: c.GetResourceHrefFilter(),
				JqExpressionFilter: c.GetJqExpressionFilter(),
			},
		},
	}
}

func (c *Condition) Clone() *Condition {
	return &Condition{
		Id:              c.Id,
		Name:            c.Name,
		Enabled:         c.Enabled,
		Owner:           c.Owner,
		ConfigurationId: c.ConfigurationId,
		ApiAccessToken:  c.ApiAccessToken,
		Timestamp:       c.Timestamp,
		Versions:        slices.Clone(c.Versions),
	}
}

func (c *Condition) GetCondition(version int) *pb.Condition {
	return &pb.Condition{
		Id:                 c.Id,
		Name:               c.Name,
		Enabled:            c.Enabled,
		Owner:              c.Owner,
		ConfigurationId:    c.ConfigurationId,
		ApiAccessToken:     c.ApiAccessToken,
		Timestamp:          c.Timestamp,
		Version:            c.Versions[version].Version,
		DeviceIdFilter:     c.Versions[version].DeviceIdFilter,
		ResourceTypeFilter: c.Versions[version].ResourceTypeFilter,
		ResourceHrefFilter: c.Versions[version].ResourceHrefFilter,
		JqExpressionFilter: c.Versions[version].JqExpressionFilter,
	}
}

func (c *Condition) Time() time.Time {
	return time.Unix(0, c.Timestamp)
}
