package pb

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
)

func (r *AppliedDeviceConfiguration_Resource) Validate() error {
	if r.GetHref() == "" {
		return errors.New("missing href")
	}
	if r.GetCorrelationId() == "" {
		return errors.New("missing correlationID")
	}
	return nil
}

func (c *AppliedDeviceConfiguration) Validate(isUpdate bool) error {
	if isUpdate || c.GetId() != "" {
		if _, err := uuid.Parse(c.GetId()); err != nil {
			return fmt.Errorf("invalid ID(%v): %w", c.GetId(), err)
		}
	}
	if c.GetOwner() == "" {
		return errors.New("missing owner")
	}
	if c.GetDeviceId() == "" {
		return errors.New("missing deviceID")
	}
	if c.GetConfigurationId() == nil || c.GetConfigurationId().GetId() == "" {
		return errors.New("invalid configurationID")
	}
	if c.GetExecutedBy() == nil {
		return errors.New("missing executedBy")
	}
	if len(c.GetResources()) == 0 {
		return errors.New("missing resources")
	}
	for _, r := range c.GetResources() {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("invalid resource: %w", err)
		}
	}
	return nil
}

func MakeRelationTo(id string, version uint64) *AppliedDeviceConfiguration_RelationTo {
	return &AppliedDeviceConfiguration_RelationTo{
		Id:      id,
		Version: version,
	}
}

func (r *AppliedDeviceConfiguration_RelationTo) Clone() *AppliedDeviceConfiguration_RelationTo {
	if r == nil {
		return nil
	}
	return &AppliedDeviceConfiguration_RelationTo{
		Id:      r.GetId(),
		Version: r.GetVersion(),
	}
}

func MakeExecutedByOnDemand() *AppliedDeviceConfiguration_OnDemand {
	return &AppliedDeviceConfiguration_OnDemand{
		OnDemand: true,
	}
}

func MakeExecutedByConditionId(conditionID string, version uint64) *AppliedDeviceConfiguration_ConditionId {
	return &AppliedDeviceConfiguration_ConditionId{
		ConditionId: &AppliedDeviceConfiguration_RelationTo{
			Id:      conditionID,
			Version: version,
		},
	}
}

func (r *AppliedDeviceConfiguration_Resource) Clone() *AppliedDeviceConfiguration_Resource {
	return &AppliedDeviceConfiguration_Resource{
		Href:            r.GetHref(),
		CorrelationId:   r.GetCorrelationId(),
		Status:          r.GetStatus(),
		ResourceUpdated: r.GetResourceUpdated().Clone(),
	}
}

func (r *AppliedDeviceConfiguration_Resource) UnmarshalBSON(data []byte) error {
	return pkgMongo.UnmarshalProtoBSON(data, r, nil)
}

func (r *AppliedDeviceConfiguration_Resource) MarshalBSON() ([]byte, error) {
	return pkgMongo.MarshalProtoBSON(r, nil)
}

func (c *AppliedDeviceConfiguration) Clone() *AppliedDeviceConfiguration {
	var executedBy isAppliedDeviceConfiguration_ExecutedBy
	if c.GetOnDemand() {
		executedBy = MakeExecutedByOnDemand()
	} else if rt := c.GetConditionId(); rt != nil {
		executedBy = MakeExecutedByConditionId(rt.GetId(), rt.GetVersion())
	}
	var resources []*AppliedDeviceConfiguration_Resource
	if len(c.GetResources()) > 0 {
		resources = make([]*AppliedDeviceConfiguration_Resource, 0, len(c.GetResources()))
		for _, r := range c.GetResources() {
			resources = append(resources, r.Clone())
		}
	}
	return &AppliedDeviceConfiguration{
		Id:              c.GetId(),
		DeviceId:        c.GetDeviceId(),
		ConfigurationId: c.GetConfigurationId().Clone(),
		ExecutedBy:      executedBy,
		Resources:       resources,
		Owner:           c.GetOwner(),
		Timestamp:       c.GetTimestamp(),
	}
}

func renameKey(json map[string]interface{}, oldKey, newKey string) {
	if v, ok := json[oldKey]; ok {
		json[newKey] = v
		delete(json, oldKey)
	}
}

func (c *AppliedDeviceConfiguration) jsonToBSONTag(json map[string]interface{}) {
	renameKey(json, "id", "_id")
	pkgMongo.ConvertStringValueToInt(json, "configurationId.version")
	pkgMongo.ConvertStringValueToInt(json, "conditionId.version")
}

func (c *AppliedDeviceConfiguration) bsonToJSONTag(json map[string]interface{}) {
	renameKey(json, "_id", "id")
}

func (c *AppliedDeviceConfiguration) UnmarshalBSON(data []byte) error {
	return pkgMongo.UnmarshalProtoBSON(data, c, c.bsonToJSONTag)
}

func (c *AppliedDeviceConfiguration) MarshalBSON() ([]byte, error) {
	return pkgMongo.MarshalProtoBSON(c, c.jsonToBSONTag)
}
