package mongodb

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type updateJSON = func(map[string]any) error

var ErrPathNotFound = errors.New("path not found")

// ConvertStringValueToInt64 converts string values to int64 in a JSON map based on provided paths.
// It iterates over the specified paths in the JSON map and converts the string values found at those paths to int64 values.
// If permitMissingPaths is set to true, missing paths in the JSON map will be ignored and the modified JSON map will be returned.
// If permitMissingPaths is set to false, an error will be returned if any of the specified paths are not found in the JSON map.
// The function returns the updated JSON map with the converted int64 values.
// If an error occurs during the conversion, the partially modified JSON map is returned along with the error.
func ConvertStringValueToInt64(jsonMap any, permitMissingPaths bool, paths ...string) (any, error) {
	for _, path := range paths {
		newMap, err := convertPath(jsonMap, permitMissingPaths, path)
		if err != nil {
			return jsonMap, err
		}
		jsonMap = newMap
	}
	return jsonMap, nil
}

func handleSlice(slice []any, permitMissingPaths bool, remainingParts []string) ([]any, error) {
	var (
		parents []any
		errs    *multierror.Error
	)

	for _, item := range slice {
		p, err := findParents(item, permitMissingPaths, remainingParts)
		if err != nil {
			errs = multierror.Append(errs, err)
		} else if p != nil {
			parents = append(parents, p...)
		}
	}

	if len(parents) == 0 {
		return nil, errs.ErrorOrNil()
	}

	return parents, errs.ErrorOrNil()
}

func findParents(current any, permitMissingPaths bool, parts []string) ([]any, error) {
	for idx, part := range parts {
		if part == "" {
			continue
		}
		switch curr := current.(type) {
		case map[string]any:
			if value, exists := curr[part]; exists {
				current = value
			} else if permitMissingPaths {
				return nil, nil
			} else {
				return nil, fmt.Errorf("path segment %s: %w", part, ErrPathNotFound)
			}
		case []any:
			if part == "" || part == "*" {
				return handleSlice(curr, permitMissingPaths, parts[idx+1:])
			}
			index, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid array index %s", part)
			}
			if index < 0 || index >= len(curr) {
				if permitMissingPaths {
					return nil, nil
				}
				return nil, fmt.Errorf("index out of range %d: %w", index, ErrPathNotFound)
			}
			current = curr[index]
		default:
			return nil, fmt.Errorf("unsupported type %T at path segment %s", current, part)
		}
	}
	return []any{current}, nil
}

var splitPathRE = regexp.MustCompile(`\.\[|\]\.|\.|\[|\]`)

func splitPath(path string) []string {
	parts := splitPathRE.Split(path, -1)
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	return cleanParts
}

func setMap(data any, permitMissingPaths bool, path string, parent map[string]any, lastPart string) (out any, err error) {
	value, exists := parent[lastPart]
	if !exists {
		if permitMissingPaths {
			return data, nil
		}
		return data, fmt.Errorf("path %s: %w", path, ErrPathNotFound)
	}
	strVal, ok := value.(string)
	if !ok {
		return data, fmt.Errorf("expected string at path %s, but found %T", path, value)
	}
	intVal, err := strconv.ParseInt(strVal, 10, 64)
	if err != nil {
		return data, fmt.Errorf("error converting string to int64 at path %s: %w", path, err)
	}
	parent[lastPart] = intVal

	return data, nil
}

func setSliceValue(data any, permitMissingPaths bool, path string, parent []any, index int) (out any, err error) {
	if index < 0 || index >= len(parent) {
		if permitMissingPaths {
			return data, nil
		}
		return data, fmt.Errorf("index out of range %d", index)
	}
	if value, ok := parent[index].(string); ok {
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return data, fmt.Errorf("error converting string to int64 at path %s: %w", path, err)
		}
		parent[index] = intVal
	} else {
		return data, fmt.Errorf("expected string at path %s, but found %T", path, parent[index])
	}
	return data, nil
}

func setSlice(data any, permitMissingPaths bool, path string, parent []any, lastPart string) (out any, err error) {
	if lastPart == "" || lastPart == "*" {
		out = data
		for i := range parent {
			out, err = setSliceValue(out, permitMissingPaths, path, parent, i)
			if err != nil {
				return out, err
			}
		}
		return out, err
	}
	index, err := strconv.Atoi(lastPart)
	if err != nil {
		return data, fmt.Errorf("invalid array index %s", lastPart)
	}
	return setSliceValue(data, permitMissingPaths, path, parent, index)
}

func setDirectValue(data any, path string) (out any, err error) {
	intVal, err := strconv.ParseInt(data.(string), 10, 64)
	if err != nil {
		return data, fmt.Errorf("error converting string to int64 at path %s: %w", path, err)
	}
	return intVal, nil
}

func convertPath(data any, permitMissingPaths bool, path string) (out any, err error) {
	var parentsRaw []any
	var lastPart string
	var parts []string
	if path == "." {
		parentsRaw = []any{data}
	} else {
		parts = splitPath(path)
		if len(parts) == 0 {
			return data, errors.New("empty path")
		}

		lastPart = parts[len(parts)-1]
		parentsRaw, err = findParents(data, permitMissingPaths, parts[:len(parts)-1])
		if err != nil {
			return data, fmt.Errorf("error finding parent for path %s: %w", path, err)
		}
	}

	out = data
	var errs *multierror.Error
	for _, parentRaw := range parentsRaw {
		switch parent := parentRaw.(type) {
		case map[string]any:
			out, err = setMap(out, permitMissingPaths, path, parent, lastPart)
			if err != nil {
				errs = multierror.Append(errs, err)
			}
		case []any:
			out, err = setSlice(out, permitMissingPaths, path, parent, lastPart)
			if err != nil {
				errs = multierror.Append(errs, err)
			}
		case string:
			out, err = setDirectValue(parent, path)
			if err != nil {
				errs = multierror.Append(errs, err)
			}
		default:
			return data, fmt.Errorf("unsupported type %T at parent path %s", parent, strings.Join(parts[:len(parts)-1], "."))
		}
	}
	return out, errs.ErrorOrNil()
}

func UnmarshalProtoBSON(data []byte, m proto.Message, update updateJSON) error {
	var obj map[string]any
	if err := bson.Unmarshal(data, &obj); err != nil {
		return err
	}
	if update != nil {
		if err := update(obj); err != nil {
			return err
		}
	}
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonData, m)
}

func MarshalProtoBSON(m proto.Message, update updateJSON) ([]byte, error) {
	data, err := protojson.Marshal(m)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	err = json.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}
	if update != nil {
		if err := update(obj); err != nil {
			return nil, err
		}
	}
	return bson.Marshal(obj)
}
