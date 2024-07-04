package jq

import (
	"fmt"

	"github.com/itchyny/gojq"
)

func EvalJQCondition(jq string, v any) (bool, error) {
	q, err := gojq.Parse(jq)
	if err != nil {
		return false, fmt.Errorf("cannot parse jq query(%v): %w", jq, err)
	}
	iter := q.Run(v)
	val, ok := iter.Next()
	if !ok {
		return false, fmt.Errorf("jq query(%v) returned no result", jq)
	}
	if result, ok := val.(bool); ok {
		return result, nil
	}
	return false, fmt.Errorf("invalid jq result: %v", val)
}
