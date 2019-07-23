package testlab

import (
	"context"
	"fmt"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/testlab/node"
	"github.com/libp2p/testlab/utils"
	"github.com/sirupsen/logrus"
)

// Deployment is a pair of a Node and a Quantity of that node to schedule in the
// cluster.
type Deployment struct {
	Name         string
	Plugin       string
	Options      utils.NodeOptions
	Quantity     int
	Dependencies []string
}

func (d *Deployment) TaskGroup(consul *capi.Client) (*napi.TaskGroup, node.PostDeployFunc, error) {
	group := napi.NewTaskGroup(d.Name, d.Quantity)
	group.Count = &d.Quantity

	plugin, err := node.GetPlugin(d.Plugin)
	if err != nil {
		return nil, nil, err
	}
	task, err := plugin.Task(consul, d.Options)
	if err != nil {
		return nil, nil, err
	}
	group.AddTask(task)
	postDeploy := func(c *capi.Client) error {
		return plugin.PostDeploy(c, d.Options)
	}
	return group, postDeploy, nil
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

func (t *Topology) Phases() ([][]*Deployment, error) {
	var phases [][]*Deployment
	scheduled := make(map[string]struct{})
	for {
		var nextPhase []*Deployment
		numScheduled := len(scheduled)
		if numScheduled == len(t.Deployments) {
			break
		}
	DeploymentLoop:
		for _, deployment := range t.Deployments {
			if _, ok := scheduled[deployment.Name]; ok {
				continue
			}
			for _, dep := range deployment.Dependencies {
				if _, ok := scheduled[dep]; !ok {
					continue DeploymentLoop
				}
			}
			nextPhase = append(nextPhase, deployment)
		}
		for _, dep := range nextPhase {
			scheduled[dep.Name] = struct{}{}
		}
		if numScheduled == len(scheduled) {
			return nil, fmt.Errorf("could not resolve dependencies")
		}
		phases = append(phases, nextPhase)
	}

	return phases, nil
}

type RenderedPhase struct {
	Job                 *napi.Job
	PostDeploymentFuncs []node.PostDeployFunc
}

func (t *Topology) JobsChan(ctx context.Context, consul *capi.Client) (<-chan RenderedPhase, <-chan struct{}, <-chan error) {
	outch := make(chan RenderedPhase)
	pullch := make(chan struct{})
	errch := make(chan error)
	closer := func() {
		close(outch)
		close(pullch)
		close(errch)
	}

	opts := t.Options
	region := opts.Region
	if opts.Region == "" {
		region = "global"
	}
	if opts.Priority == 0 {
		opts.Priority = 50
	}

	phases, err := t.Phases()
	if err != nil {
		errch <- err
		closer()
		return outch, pullch, errch
	}

	go func() {
		defer closer()
		for i, phase := range phases {
			select {
			case pullch <- struct{}{}:
				postDeployFuncs := make([]node.PostDeployFunc, len(phase))
				name := fmt.Sprintf("%s_phase_%d", t.Name, i)
				job := napi.NewServiceJob(name, name, region, opts.Priority)
				job.Datacenters = opts.Datacenters
				for e, deployment := range phase {
					logrus.Infof("deployment: %s", deployment.Name)
					group, postDeploy, err := deployment.TaskGroup(consul)
					if err != nil {
						errch <- err
						return
					}
					job.AddTaskGroup(group)
					postDeployFuncs[e] = postDeploy
				}
				outch <- RenderedPhase{job, postDeployFuncs}
			case <-ctx.Done():
				errch <- ctx.Err()
				return
			}
		}
	}()

	return outch, pullch, errch
}

func (t *Topology) Jobs(consul *capi.Client) ([]*napi.Job, [][]node.PostDeployFunc, error) {
	opts := t.Options
	region := opts.Region
	if opts.Region == "" {
		region = "global"
	}
	if opts.Priority == 0 {
		opts.Priority = 50
	}

	phases, err := t.Phases()
	if err != nil {
		return nil, nil, err
	}

	jobs := make([]*napi.Job, len(phases))
	postDeployFuncs := make([][]node.PostDeployFunc, len(phases))
	for i, phase := range phases {
		phasePostDeployFuncs := make([]node.PostDeployFunc, len(phase))
		name := fmt.Sprintf("%s_phase_%d", t.Name, i)
		job := napi.NewServiceJob(name, name, region, opts.Priority)
		job.Datacenters = opts.Datacenters
		for e, deployment := range phase {
			group, postDeploy, err := deployment.TaskGroup(consul)
			if err != nil {
				return nil, nil, err
			}
			job.AddTaskGroup(group)
			phasePostDeployFuncs[e] = postDeploy
		}
		jobs[i] = job
		postDeployFuncs[i] = phasePostDeployFuncs
	}

	return jobs, postDeployFuncs, nil
}
