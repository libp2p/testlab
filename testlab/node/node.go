package node

import (
	"fmt"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/testlab/node/ipfs"
	"github.com/libp2p/testlab/testlab/node/p2pd"
	"github.com/libp2p/testlab/testlab/node/prometheus"
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
		"p2pd":       new(p2pd.Node),
		"ipfs":       new(ipfs.Node),
		"scenario":   new(scenario.Node),
		"prometheus": new(prometheus.Node),
	}
}

type PostDeployFunc func(*capi.Client) error

// Node is an incredibly simple interface describing plugins that will generate
// nomad tasks. For now, this is left as an interface so plugin implementors can
// include instantiation logic.
type Node interface {
	TaskGroups(consul *capi.Client, name string, quantity int, options utils.NodeOptions) ([]*napi.TaskGroup, error)
	PostDeploy(*capi.Client, utils.NodeOptions) error
}
