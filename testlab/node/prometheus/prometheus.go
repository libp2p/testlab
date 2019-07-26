package prometheus

import (
	"time"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/utils"
)

// Node is the struct that builds prometheus tasks.
type Node struct{}

var config = `---
global:
  scrape_interval:     5s
  evaluation_interval: 5s

scrape_configs:
  - job_name: 'testlab_metrics'

    consul_sd_configs:
    - server: '{{ env "CONSUL_HTTP_ADDR" }}'
      datacenter: '{{ or (env "CONSUL_DATACENTER") "" }}'
      services: ['metrics']

    scrape_interval: 5s
`

const MetricsServiceName = "metrics"

func (n *Node) PostDeploy(consul *capi.Client, options utils.NodeOptions) error {
	return nil
}

// TaskGroups creates a nomad task specification for our prometheus metrics collector
func (n *Node) TaskGroups(consul *capi.Client, name string, quantity int, options utils.NodeOptions) ([]*napi.TaskGroup, error) {
	group := napi.NewTaskGroup(name, 1)
	task := napi.NewTask("prometheus", "docker")

	res := napi.DefaultResources()
	res.Networks = []*napi.NetworkResource{
		{
			DynamicPorts: []napi.Port{
				napi.Port{Label: "prometheus"},
			},
		},
	}
	mem := 1000

	if memOpt, ok := options.Int("Memory"); ok {
		mem = memOpt
	}

	res.MemoryMB = &mem
	task.Resources = res

	task.Env = make(map[string]string)
	utils.AddConsulEnvToTask(task)
	tpl := &napi.Template{
		EmbeddedTmpl: &config,
		DestPath:     utils.StringPtr("local/prometheus.yml"),
	}
	task.Templates = append(task.Templates, tpl)

	task.SetConfig("image", "prom/prometheus:latest")
	task.SetConfig("volumes", []string{
		"local/prometheus.yml:/etc/prometheus/prometheus.yml",
	})
	task.SetConfig("port_map", []interface{}{
		map[string]interface{}{"prometheus": 9090},
	})

	svc := &napi.Service{
		Name:      "prometheus",
		PortLabel: "prometheus",
		Checks: []napi.ServiceCheck{
			napi.ServiceCheck{
				Name:     "prometheus port alive",
				Type:     "http",
				Path:     "/-/healthy",
				Interval: 10 * time.Second,
				Timeout:  2 * time.Second,
			},
		},
	}
	task.Services = append(task.Services, svc)

	group.AddTask(task)
	return []*napi.TaskGroup{group}, nil
}
