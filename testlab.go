package testlab

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	napi "github.com/hashicorp/nomad/api"
	"github.com/sirupsen/logrus"
)

type Node interface {
	Task() *napi.Task
}

type P2pdNode struct {
}

var _ Node = (*P2pdNode)(nil)

func (n *P2pdNode) Task() *napi.Task {
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

	return task
}

// Deployment is a pair of a Node and a Quantity of that node to schedule in the
// cluster.
type Deployment struct {
	Name string
	// Node     Node
	Quantity int
}

func (d *Deployment) TaskGroup() *napi.TaskGroup {
	group := napi.NewTaskGroup(d.Name, d.Quantity)
	group.Count = &d.Quantity
	var node P2pdNode
	group.AddTask(node.Task())
	return group
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
	// Deployments details the different deployments to schedule on the nomad
	// cluster.
	Deployments []*Deployment
}

func (t *Topology) Job() *napi.Job {
	opts := t.Options
	region := opts.Region
	if opts.Region == "" {
		region = "global"
	}

	job := napi.NewServiceJob(t.Name, t.Name, region, opts.Priority)
	job.Datacenters = opts.Datacenters

	for _, deployment := range t.Deployments {
		job.AddTaskGroup(deployment.TaskGroup())
	}

	return job
}

type TestLab struct {
	path       string
	nomad      *napi.Client
	deployment string
}

// NewTestlab initiates a testlab, with a path to the current state of the
// testlab as well as a configuration for contacting the nomad cluster.
func NewTestlab(path string, nomadConfig *napi.Config) (*TestLab, error) {
	if nomadConfig == nil {
		nomadConfig = napi.DefaultConfig()
	}

	client, err := napi.NewClient(nomadConfig)
	if err != nil {
		return nil, err
	}

	pathStat, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 755)
		if err != nil {
			return nil, fmt.Errorf("creating root directory: %s", err)
		}
	} else if !pathStat.IsDir() {
		return nil, fmt.Errorf("expected root (%s) to be directory", path)
	}

	deploymentPath := filepath.Join(path, "deployment")
	_, err = os.Stat(deploymentPath)
	var deployment string
	if err == nil {
		bs, err := ioutil.ReadFile(deploymentPath)
		if err != nil {
			logrus.Fatalf("failed to read deployment id <%s>, deleting", err)
		}
		deployment = strings.TrimSpace(string(bs))
	}

	testLab := &TestLab{
		path:       path,
		nomad:      client,
		deployment: deployment,
	}
	return testLab, nil
}

// Clear stops a running deployment
func (t *TestLab) Clear() error {
	if t.deployment == "" {
		logrus.Info("no existing deployment to tear down")
		return nil
	}

	return t.Deregister(t.deployment)
}

func (t *TestLab) Deregister(jobID string) error {
	evalID, _, err := t.nomad.Jobs().Deregister(jobID, false, nil)
	if err != nil {
		logrus.Errorf("failed to teardown job %s: %s", jobID, err)
		return err
	}
	logrus.Infof("cleared job id %s in evaluation %s", jobID, evalID)
	os.Remove(filepath.Join(t.path, "deployment"))
	return err
}

func (t *TestLab) Start(topology *Topology) error {
	wopts := &napi.WriteOptions{}
	resp, _, err := t.nomad.Jobs().Register(topology.Job(), wopts)
	if err == nil {
		logrus.Infof("rendering topology in evaluation id %s took %s", resp.EvalID, resp.RequestTime.String())
		return err
	}
	err = ioutil.WriteFile(filepath.Join(t.path, "deployment"), []byte(topology.Name), 644)
	return err
}
