package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMergeYamlNodes(t *testing.T) {
	type args struct {
		node1 *yaml.Node
		node2 *yaml.Node
	}
	tests := []struct {
		name    string
		args    args
		want    *yaml.Node
		wantErr bool
	}{
		{
			name: "MergeTwoScalarNodes",
			args: args{
				node1: &yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "key1"},
						{Kind: yaml.ScalarNode, Value: "value1"},
					},
				},
				node2: &yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "key2"},
						{Kind: yaml.ScalarNode, Value: "value2"},
					},
				},
			},
			want: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key1"},
					{Kind: yaml.ScalarNode, Value: "value1"},
					{Kind: yaml.ScalarNode, Value: "key2"},
					{Kind: yaml.ScalarNode, Value: "value2"},
				},
			},
		},
		{
			name: "MergeTwoSequenceNodes",
			args: args{
				node1: &yaml.Node{
					Kind: yaml.SequenceNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "value1"},
					},
				},
				node2: &yaml.Node{
					Kind: yaml.SequenceNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "value2"},
						{Kind: yaml.ScalarNode, Value: "value3"},
					},
				},
			},
			want: &yaml.Node{
				Kind: yaml.SequenceNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "value1"},
					{Kind: yaml.ScalarNode, Value: "value2"},
					{Kind: yaml.ScalarNode, Value: "value3"},
				},
			},
		},
		{
			name: "MergeTwoMappingNodesWithOverlappingKeys",
			args: args{
				node1: &yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "key1"},
						{Kind: yaml.ScalarNode, Value: "value1"},
						{Kind: yaml.ScalarNode, Value: "key2"},
						{Kind: yaml.ScalarNode, Value: "value2"},
					},
				},
				node2: &yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "key2"},
						{Kind: yaml.ScalarNode, Value: "value3"},
						{Kind: yaml.ScalarNode, Value: "key3"},
						{Kind: yaml.ScalarNode, Value: "value4"},
					},
				},
			},
			want: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key1"},
					{Kind: yaml.ScalarNode, Value: "value1"},
					{Kind: yaml.ScalarNode, Value: "key2"},
					{Kind: yaml.ScalarNode, Value: "value3"},
					{Kind: yaml.ScalarNode, Value: "key3"},
					{Kind: yaml.ScalarNode, Value: "value4"},
				},
			},
		},
		{
			args: args{
				node1: &yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "key1"},
						{Kind: yaml.ScalarNode, Value: "value1"},
					},
				},
				node2: &yaml.Node{
					Kind: yaml.SequenceNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "value2"},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// merge the nodes
			got, err := MergeYamlNodes(tt.args.node1, tt.args.node2)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
