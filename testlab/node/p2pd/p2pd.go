package p2pd

import (
	"fmt"

	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/utils"
	"github.com/sirupsen/logrus"
)

type Node struct{}

func (n *Node) Task(options utils.NodeOptions) (*napi.Task, error) {
	task := napi.NewTask("p2pd", "exec")
	command := "/usr/local/bin/p2pd"
	args := []string{
		"-listen", "/ip4/${NOMAD_IP_p2pd}/tcp/${NOMAD_PORT_p2pd}",
		"-hostAddrs", "/ip4/${NOMAD_IP_libp2p}/tcp/${NOMAD_PORT_libp2p}",
	}

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

	p2pdSvc := &napi.Service{
		Name:      "p2pd",
		PortLabel: "p2pd",
	}
	task.Services = append(task.Services, p2pdSvc)

	url := ""

	if cid, ok := options.String("Cid"); ok {
		url = fmt.Sprintf("https://gateway.ipfs.io/ipfs/%s", cid)
	}

	if urlOpt, ok := options.String("Fetch"); ok {
		url = urlOpt
	}

	if url != "" {
		task.Artifacts = []*napi.TaskArtifact{
			&napi.TaskArtifact{
				GetterSource: utils.StringPtr(url),
				RelativeDest: utils.StringPtr("p2pd"),
			},
		}
		command = "p2pd"
	}

	if service, ok := options.String("Service"); ok {
		if service == "p2pd" {
			logrus.Error("p2pd already exports service \"p2pd\"")
		} else {
			svc := &napi.Service{
				Name:      service,
				PortLabel: "libp2p",
			}
			task.Services = append(task.Services, svc)
		}
	}

	if bootstrap, ok := options["Bootstrap"]; ok {
		tmpl := fmt.Sprintf("BOOTSTRAP_PEERS={{range $index, $service := service \"%s\"}}{{if ne $index 0}},{{end}}/ip4/{{$service.Address}}/tcp/{{$service.Port}}{{end}}", bootstrap)
		env := true
		template := &napi.Template{
			EmbeddedTmpl: &tmpl,
			DestPath:     utils.StringPtr("bootstrap_peers.env"),
			Envvars:      &env,
		}
		task.Templates = append(task.Templates, template)
		args = append(args, "-bootstrapPeers", "${BOOTSTRAP_PEERS}")
	}

	task.SetConfig("command", command)
	task.SetConfig("args", args)

	return task, nil
}
