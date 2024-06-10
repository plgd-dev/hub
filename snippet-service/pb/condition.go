package pb

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/strings"
)

func checkConfigurationId(c string, isUpdate bool) error {
	if isUpdate && c == "" {
		// in this case the update will keep the configuration ID already in the database
		return nil
	}
	if _, err := uuid.Parse(c); err != nil {
		return fmt.Errorf("invalid configuration ID(%v): %w", c, err)
	}
	return nil
}

func (c *Condition) Validate(isUpdate bool) error {
	if isUpdate || c.GetId() != "" {
		if _, err := uuid.Parse(c.GetId()); err != nil {
			return fmt.Errorf("invalid ID(%v): %w", c.GetId(), err)
		}
	}
	if err := checkConfigurationId(c.GetConfigurationId(), isUpdate); err != nil {
		return err
	}
	if c.GetOwner() == "" {
		return errors.New("missing owner")
	}
	return nil
}

func (c *Condition) Normalize() {
	c.DeviceIdFilter = strings.Unique(c.GetDeviceIdFilter())
	c.ResourceTypeFilter = strings.Unique(c.GetResourceTypeFilter())
	c.ResourceHrefFilter = strings.Unique(c.GetResourceHrefFilter())
}

func (c *Condition) Clone() *Condition {
	return &Condition{
		Id:                 c.GetId(),
		Name:               c.GetName(),
		Enabled:            c.GetEnabled(),
		Owner:              c.GetOwner(),
		ConfigurationId:    c.GetConfigurationId(),
		ApiAccessToken:     c.GetApiAccessToken(),
		Timestamp:          c.GetTimestamp(),
		Version:            c.GetVersion(),
		DeviceIdFilter:     slices.Clone(c.GetDeviceIdFilter()),
		ResourceTypeFilter: slices.Clone(c.GetResourceTypeFilter()),
		ResourceHrefFilter: slices.Clone(c.GetResourceHrefFilter()),
		JqExpressionFilter: c.GetJqExpressionFilter(),
	}
}
