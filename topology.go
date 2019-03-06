package testlab

import (
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/testlab/node"
)

// Deployment is a pair of a Node and a Quantity of that node to schedule in the
// cluster.
type Deployment struct {
	Name     string
	Plugin   string
	Options  utils.NodeOptions
	Quantity int
}

func (d *Deployment) TaskGroup() (*napi.TaskGroup, error) {
	group := napi.NewTaskGroup(d.Name, d.Quantity)
	group.Count = &d.Quantity

	node, err := node.GetPlugin(d.Plugin)
	if err != nil {
		return nil, err
	}
	task, err := node.Task(d.Options)
	if err != nil {
		return nil, err
	}
	group.AddTask(task)
	return group, nil
}

type TopologyOptions struct {
	Region      string
	Priority    int
	Datacenters []string
}

type Topology struct {
	Options *TopologyOptions
	// Name will be translated into a nomad job
	Name string
	// Deployments details the different node types to schedule on the nomad
	// cluster.
	Deployments []*Deployment
}

func (t *Topology) Job() (*napi.Job, error) {
	opts := t.Options
	region := opts.Region
	if opts.Region == "" {
		region = "global"
	}

	job := napi.NewServiceJob(t.Name, t.Name, region, opts.Priority)
	job.Datacenters = opts.Datacenters

	for _, deployment := range t.Deployments {
		group, err := deployment.TaskGroup()
		if err != nil {
			return nil, err
		}
		job.AddTaskGroup(group)
	}

	return job, nil
}
