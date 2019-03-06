package node

import (
	"fmt"

	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/testlab/node/p2pd"
	"github.com/libp2p/testlab/testlab/node/scenario"
	"github.com/libp2p/testlab/utils"
)

var Plugins map[string]Node

func GetPlugin(name string) (Node, error) {
	plugin, ok := Plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin \"%s\" not registered", name)
	}
	return plugin, nil
}

func init() {
	Plugins = map[string]Node{
		"p2pd":     new(p2pd.P2pdNode),
		"scenario": new(scenario.ScenarioNode),
	}
}

// Node is an incredibly simple interface describing plugins that will generate
// nomad tasks. For now, this is left as an interface so plugin implementors can
// include instantiation logic.
type Node interface {
	Task(options utils.NodeOptions) (*napi.Task, error)
}
