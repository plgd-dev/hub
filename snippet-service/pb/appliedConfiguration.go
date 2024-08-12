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
	if r.GetStatus() == AppliedConfiguration_Resource_UNSPECIFIED {
		return errors.New("invalid status")
	}
	return nil
}

func (c *AppliedConfiguration) Validate() error {
	if c.GetId() != "" {
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

func MakeLinkedTo(id string, version uint64) *AppliedConfiguration_LinkedTo {
	return &AppliedConfiguration_LinkedTo{
		Id:      id,
		Version: version,
	}
}

func (r *AppliedConfiguration_LinkedTo) Clone() *AppliedConfiguration_LinkedTo {
	if r == nil {
		return nil
	}
	return &AppliedConfiguration_LinkedTo{
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
		ConditionId: &AppliedConfiguration_LinkedTo{
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
		ValidUntil:      r.GetValidUntil(),
	}
}

func (r *AppliedConfiguration_Resource) jsonToBSONTag(json map[string]interface{}) error {
	if _, err := pkgMongo.ConvertStringValueToInt64(json, true, "."+ValidUntil); err != nil {
		return fmt.Errorf("cannot convert .validUntil to int64: %w", err)
	}
	return nil
}

func (r *AppliedConfiguration_Resource) MarshalBSON() ([]byte, error) {
	return pkgMongo.MarshalProtoBSON(r, r.jsonToBSONTag)
}

func (c *AppliedConfiguration) CloneExecutedBy() isAppliedConfiguration_ExecutedBy {
	var executedBy isAppliedConfiguration_ExecutedBy
	if c.GetOnDemand() {
		executedBy = MakeExecutedByOnDemand()
	} else if rt := c.GetConditionId(); rt != nil {
		executedBy = MakeExecutedByConditionId(rt.GetId(), rt.GetVersion())
	}
	return executedBy
}

func (c *AppliedConfiguration) CloneAppliedConfiguration_Resources() []*AppliedConfiguration_Resource {
	var resources []*AppliedConfiguration_Resource
	if len(c.GetResources()) > 0 {
		resources = make([]*AppliedConfiguration_Resource, 0, len(c.GetResources()))
		for _, r := range c.GetResources() {
			resources = append(resources, r.Clone())
		}
	}
	return resources
}

func (c *AppliedConfiguration) Clone() *AppliedConfiguration {
	return &AppliedConfiguration{
		Id:              c.GetId(),
		DeviceId:        c.GetDeviceId(),
		ConfigurationId: c.GetConfigurationId().Clone(),
		ExecutedBy:      c.CloneExecutedBy(),
		Resources:       c.CloneAppliedConfiguration_Resources(),
		Owner:           c.GetOwner(),
		Timestamp:       c.GetTimestamp(),
	}
}

func (c *AppliedConfiguration) jsonToBSONTag(json map[string]interface{}) error {
	if _, err := pkgMongo.ConvertStringValueToInt64(json, true, "."+ConfigurationIDKey+"."+VersionKey); err != nil {
		return fmt.Errorf("cannot convert configurationId.version to int64: %w", err)
	}
	if _, err := pkgMongo.ConvertStringValueToInt64(json, true, "."+ConditionIDKey+"."+VersionKey); err != nil {
		return fmt.Errorf("cannot convert conditionId.version to int64: %w", err)
	}
	if _, err := pkgMongo.ConvertStringValueToInt64(json, true, "."+ResourcesKey+".[*]."+ValidUntil); err != nil {
		return fmt.Errorf("cannot convert resources.validUntil to int64: %w", err)
	}
	return nil
}

func (c *AppliedConfiguration) MarshalBSON() ([]byte, error) {
	return pkgMongo.MarshalProtoBSON(c, c.jsonToBSONTag)
}
