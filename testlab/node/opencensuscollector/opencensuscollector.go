package opencensuscollector

import (
	"fmt"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/utils"
)

// Node defines how to construct a prometheus metrics collector
type Node struct{}

func (n *Node) configTemplate() *string {
	cfg := capi.DefaultConfig()
	var tpl = `receivers:
  opencensus:
    address: '{{ env "NOMAD_ADDR_opencensus "}}'

  prometheus:
    config:
      scrape_configs:
        - job_name: 'testlab'
		  scrape_interval: 5s
		  consul_sd_configs:
			- server: '%s'
			  token: '%s'
			  datacenter: '%s'
			  username: '%s'
			  password: '%s'
			  services:
			    - 'metrics'

`
	tpl = fmt.Sprintf(
		tpl,
		cfg.Address,
		cfg.Token,
		cfg.Datacenter,
		cfg.HttpAuth.Username,
		cfg.HttpAuth.Password,
	)
	return &tpl
}

// Task generates a nomad task to run an opencensus collector
func (n *Node) Task(options utils.NodeOptions) (*napi.Task, error) {
	task := napi.NewTask("opencensus", "exec")

	task.SetConfig("command", "occollector")
	task.SetConfig("args", []string{
		"--config",
		"config",
		"--port",
		`{{env "NOMAD_PORT_metrics"}}`,
	})

	task.Templates = []*napi.Template{
		&napi.Template{
			EmbeddedTmpl: n.configTemplate(),
			DestPath:     utils.StringPtr("config"),
		},
	}

	res := napi.DefaultResources()
	res.Networks = []*napi.NetworkResource{
		&napi.NetworkResource{
			DynamicPorts: []napi.Port{
				napi.Port{Label: "metrics"},
				napi.Port{Label: "opencensus"},
			},
		},
	}
	task.Require(res)

	metricsSvc := &napi.Service{
		Name:      "metrics",
		PortLabel: "metrics",
	}
	opencensusSvc := &napi.Service{
		Name:      "opencensus-collector",
		PortLabel: "opencensus",
	}

	task.Services = []*napi.Service{
		metricsSvc,
		opencensusSvc,
	}

	return task, nil
}
