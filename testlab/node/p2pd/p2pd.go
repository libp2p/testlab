package p2pd

import (
	"fmt"

	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/utils"
)

type P2pdNode struct{}

func (n *P2pdNode) Task(options map[string]string) *napi.Task {
	task := napi.NewTask("p2pd", "exec")
	task.SetConfig("command", "/usr/local/bin/p2pd")
	task.SetConfig("args", []string{
		"-listen", "/ip4/${NOMAD_IP_p2pd}/tcp/${NOMAD_PORT_p2pd}",
	})
	res := napi.DefaultResources()
	res.Networks = []*napi.NetworkResource{
		&napi.NetworkResource{
			DynamicPorts: []napi.Port{
				napi.Port{Label: "libp2p"},
				napi.Port{Label: "p2pd"},
			},
		},
	}
	task.Require(res)

	if cid, ok := options["Cid"]; ok {
		url := fmt.Sprintf("https://gateway.ipfs.io/ipfs/%s", cid)
		task.Artifacts = []*napi.TaskArtifact{
			&napi.TaskArtifact{
				GetterSource: utils.StringPtr(url),
				RelativeDest: utils.StringPtr("p2pd"),
			},
		}
		task.SetConfig("command", "p2pd")
	}

	return task
}
