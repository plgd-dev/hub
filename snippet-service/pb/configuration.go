package pb

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

func (cr *Configuration_Resource) Clone() *Configuration_Resource {
	if cr == nil {
		return nil
	}
	return &Configuration_Resource{
		Href:       cr.GetHref(),
		Content:    cr.GetContent().Clone(),
		TimeToLive: cr.GetTimeToLive(),
	}
}

func (c *Configuration) Validate(isUpdate bool) error {
	if isUpdate || c.GetId() != "" {
		if _, err := uuid.Parse(c.GetId()); err != nil {
			return fmt.Errorf("invalid ID(%v): %w", c.GetId(), err)
		}
	}
	if c.GetOwner() == "" {
		return errors.New("missing owner")
	}
	if len(c.GetResources()) == 0 {
		return errors.New("missing resources")
	}
	return nil
}

func normalizeResources(resources []*Configuration_Resource) []*Configuration_Resource {
	resources = slices.CompactFunc(resources, func(i, j *Configuration_Resource) bool {
		return i.GetHref() == j.GetHref()
	})
	slices.SortFunc(resources, func(i, j *Configuration_Resource) int {
		return strings.Compare(i.GetHref(), j.GetHref())
	})
	return resources
}

func (c *Configuration) Normalize() {
	c.Resources = normalizeResources(c.GetResources())
}

func (c *Configuration) Clone() *Configuration {
	cfg := &Configuration{
		Id:        c.GetId(),
		Version:   c.GetVersion(),
		Name:      c.GetName(),
		Owner:     c.GetOwner(),
		Timestamp: c.GetTimestamp(),
	}
	for _, r := range c.GetResources() {
		cfg.Resources = append(cfg.Resources, r.Clone())
	}
	return cfg
}
