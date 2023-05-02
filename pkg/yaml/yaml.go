package yaml

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func MergeYamlNodes(node1, node2 *yaml.Node) (*yaml.Node, error) {
	if node1 == nil {
		return node2, nil
	}
	if node2 == nil {
		return node1, nil
	}
	if node1.Kind != node2.Kind {
		return nil, fmt.Errorf("unable to merge nodes of different kinds: %v and %v", node1.Kind, node2.Kind)
	}
	switch node1.Kind {
	case yaml.SequenceNode:
		content := make([]*yaml.Node, 0, len(node1.Content)+len(node2.Content))
		content = append(content, node1.Content...)
		content = append(content, node2.Content...)
		return &yaml.Node{Kind: yaml.SequenceNode, Content: content}, nil
	case yaml.MappingNode:
		result := &yaml.Node{Kind: yaml.MappingNode}
		mergedKeys := make(map[string]int, len(node1.Content)/2+len(node2.Content)/2)
		for i := 0; i < len(node1.Content); i += 2 {
			key := node1.Content[i].Value
			value := node1.Content[i+1]
			result.Content = append(result.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
			mergedKeys[key] = len(result.Content) - 1
		}
		for i := 0; i < len(node2.Content); i += 2 {
			key := node2.Content[i].Value
			value := node2.Content[i+1]
			if _, ok := mergedKeys[key]; !ok {
				result.Content = append(result.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
			} else {
				newVal, err := MergeYamlNodes(result.Content[mergedKeys[key]], value)
				if err != nil {
					return nil, err
				}
				result.Content[mergedKeys[key]] = newVal
			}
		}
		return result, nil
	default:
		return node2, nil
	}
}
