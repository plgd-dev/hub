package pb

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

func (c *Configuration) ValidateAndNormalize() error {
	if _, err := uuid.Parse(c.GetId()); err != nil {
		return fmt.Errorf("invalid configuration ID(%v): %w", c.GetId(), err)
	}
	if len(c.GetResources()) == 0 {
		return errors.New("invalid configuration resources")
	}
	if c.GetOwner() == "" {
		return errors.New("empty configuration owner")
	}
	slices.SortFunc(c.GetResources(), func(i, j *Configuration_Resource) int {
		return strings.Compare(i.GetHref(), j.GetHref())
	})
	return nil
}
