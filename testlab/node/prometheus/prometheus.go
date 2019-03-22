package prometheus

import (
	"fmt"
	"time"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/utils"
)

// Node is the struct that builds prometheus tasks.
type Node struct{}

func config() string {
	cfg := capi.DefaultConfig()
	var tpl = `---
global:
  scrape_interval:     5s
  evaluation_interval: 5s

scrape_configs:
  - job_name: 'testlab_metrics'

    consul_sd_configs:
    - server: '%s'
      datacenter: '%s'
      services: ['metrics']

    relabel_configs:
    - source_labels: ['__meta_consul_tags']
      regex: '(.*)http(.*)'
      action: keep

    scrape_interval: 5s
`
	return fmt.Sprintf(
		tpl,
		cfg.Address,
		cfg.Datacenter,
	)
}

// Task creates a nomad task specification for our prometheus metrics collector
func (n *Node) Task(opts utils.NodeOptions) (*napi.Task, error) {
	task := napi.NewTask("prometheus", "docker")

	res := napi.DefaultResources()
	res.Networks = []*napi.NetworkResource{
		&napi.NetworkResource{
			DynamicPorts: []napi.Port{
				napi.Port{Label: "prometheus"},
			},
		},
	}
	task.Resources = res

	cfg := config()
	tpl := &napi.Template{
		EmbeddedTmpl: &cfg,
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

	return task, nil
}
