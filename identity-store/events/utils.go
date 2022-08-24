package events

import (
	"strings"

	"github.com/google/uuid"
)

func OwnerToUUID(owner string) string {
	if u, err := uuid.Parse(owner); err == nil && u != uuid.Nil {
		return owner
	}
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(owner)).String()
}

func WithOwner(owner string) func(values map[string]string) {
	return func(values map[string]string) {
		if owner == "*" {
			values[OwnerIdKey] = owner
			return
		}
		values[OwnerIdKey] = OwnerToUUID(owner)
	}
}

func WithEventType(eventType string) func(values map[string]string) {
	return func(values map[string]string) {
		values[EventTypeKey] = eventType
	}
}

func ToSubject(template string, opts ...func(values map[string]string)) string {
	values := make(map[string]string)
	for _, o := range opts {
		o(values)
	}
	for key, val := range values {
		template = strings.ReplaceAll(template, "{"+key+"}", val)
	}
	return template
}
