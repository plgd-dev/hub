package pb

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
)

func (r *AppliedConfiguration_Resource) Validate() error {
	if r.GetHref() == "" {
		return errors.New("missing href")
	}
	if r.GetCorrelationId() == "" {
		return errors.New("missing correlationID")
	}
	return nil
}

func (c *AppliedConfiguration) Validate(isUpdate bool) error {
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
	if c.GetConfigurationId().GetId() == "" {
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

func MakeRelationTo(id string, version uint64) *AppliedConfiguration_RelationTo {
	return &AppliedConfiguration_RelationTo{
		Id:      id,
		Version: version,
	}
}

func (r *AppliedConfiguration_RelationTo) Clone() *AppliedConfiguration_RelationTo {
	if r == nil {
		return nil
	}
	return &AppliedConfiguration_RelationTo{
		Id:      r.GetId(),
		Version: r.GetVersion(),
	}
}

func MakeExecutedByOnDemand() *AppliedConfiguration_OnDemand {
	return &AppliedConfiguration_OnDemand{
		OnDemand: true,
	}
}

func MakeExecutedByConditionId(conditionID string, version uint64) *AppliedConfiguration_ConditionId {
	return &AppliedConfiguration_ConditionId{
		ConditionId: &AppliedConfiguration_RelationTo{
			Id:      conditionID,
			Version: version,
		},
	}
}

func (r *AppliedConfiguration_Resource) Clone() *AppliedConfiguration_Resource {
	return &AppliedConfiguration_Resource{
		Href:            r.GetHref(),
		CorrelationId:   r.GetCorrelationId(),
		Status:          r.GetStatus(),
		ResourceUpdated: r.GetResourceUpdated().Clone(),
	}
}

func (r *AppliedConfiguration_Resource) UnmarshalBSON(data []byte) error {
	return pkgMongo.UnmarshalProtoBSON(data, r, nil)
}

func (r *AppliedConfiguration_Resource) MarshalBSON() ([]byte, error) {
	return pkgMongo.MarshalProtoBSON(r, nil)
}

func (c *AppliedConfiguration) Clone() *AppliedConfiguration {
	var executedBy isAppliedConfiguration_ExecutedBy
	if c.GetOnDemand() {
		executedBy = MakeExecutedByOnDemand()
	} else if rt := c.GetConditionId(); rt != nil {
		executedBy = MakeExecutedByConditionId(rt.GetId(), rt.GetVersion())
	}
	var resources []*AppliedConfiguration_Resource
	if len(c.GetResources()) > 0 {
		resources = make([]*AppliedConfiguration_Resource, 0, len(c.GetResources()))
		for _, r := range c.GetResources() {
			resources = append(resources, r.Clone())
		}
	}
	return &AppliedConfiguration{
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

func (c *AppliedConfiguration) jsonToBSONTag(json map[string]interface{}) {
	renameKey(json, "id", "_id")
	pkgMongo.ConvertStringValueToInt(json, "configurationId.version")
	pkgMongo.ConvertStringValueToInt(json, "conditionId.version")
}

func (c *AppliedConfiguration) bsonToJSONTag(json map[string]interface{}) {
	renameKey(json, "_id", "id")
}

func (c *AppliedConfiguration) UnmarshalBSON(data []byte) error {
	return pkgMongo.UnmarshalProtoBSON(data, c, c.bsonToJSONTag)
}

func (c *AppliedConfiguration) MarshalBSON() ([]byte, error) {
	return pkgMongo.MarshalProtoBSON(c, c.jsonToBSONTag)
}
