package testlab

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/sirupsen/logrus"
)

// TestLab is the main entrypoint for manipulating the test cluster.
type TestLab struct {
	path           string
	nomad          *napi.Client
	consul         *capi.Client
	deploymentPath string
	deployments    []string
}

// NewTestlab initiates a testlab, with a path to the current state of the
// testlab as well as a configuration for contacting the nomad cluster. If nil,
// nomadConfig will be populated with the defaults.
func NewTestlab(path string) (*TestLab, error) {
	consulConfig := capi.DefaultConfig()
	consul, err := capi.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
	nomadConfig := napi.DefaultConfig()
	nomad, err := napi.NewClient(nomadConfig)
	if err != nil {
		return nil, err
	}

	pathStat, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return nil, fmt.Errorf("creating root directory: %s", err)
		}
	} else if !pathStat.IsDir() {
		return nil, fmt.Errorf("expected root (%s) to be directory", path)
	}

	deploymentPath := filepath.Join(path, "deployment")
	_, err = os.Stat(deploymentPath)
	var deployments []string
	if err == nil {
		bs, err := ioutil.ReadFile(deploymentPath)
		if err != nil {
			logrus.Fatalf("failed to read deployment id <%s>, deleting", err)
		}
		deployments = strings.Split(strings.TrimSpace(string(bs)), "\n")
	}

	testLab := &TestLab{
		path:           path,
		nomad:          nomad,
		consul:         consul,
		deploymentPath: deploymentPath,
		deployments:    deployments,
	}
	return testLab, nil
}

// Clear stops a running deployment
func (t *TestLab) Clear() error {
	if t.deployments == nil {
		logrus.Info("no existing deployment to tear down")
		return nil
	}

	for _, deployment := range t.deployments {
		evalID, _, err := t.nomad.Jobs().Deregister(deployment, false, nil)
		if err != nil {
			logrus.Errorf("deregistering deployment: %s", err)
		} else {
			logrus.Infof("deregistered job %s in evaluation %s", deployment, evalID)
		}
	}

	return os.Remove(t.deploymentPath)
}

func (t *TestLab) WaitEval(evalID string) error {
	for {
		info, _, err := t.nomad.Evaluations().Info(evalID, nil)
		if err != nil {
			return err
		}
		if info.Status != "complete" {
			time.Sleep(time.Second)
			continue
		}
		evalInfo, _, err := t.nomad.Evaluations().Info(evalID, nil)
		if err != nil {
			return err
		}
		running := true
		for _, num := range evalInfo.QueuedAllocations {
			if num > 0 {
				running = false
				break
			}
		}
		allocInfos, _, err := t.nomad.Evaluations().Allocations(evalID, nil)
		if err != nil {
			return err
		}
		for _, alloc := range allocInfos {
			if alloc.ClientStatus != "running" {
				running = false
			}
		}
		if running {
			break
		}
		time.Sleep(time.Second * 5)
	}

	return nil
}

func (t *TestLab) Start(topology *Topology) error {
	jobs, postDeployFuncs, err := topology.Jobs()
	if err != nil {
		return err
	}
	deploymentFile, err := os.OpenFile(t.deploymentPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer deploymentFile.Close()
	for i, job := range jobs {
		logrus.Infof("scheduling phase %d...", i)
		resp, _, err := t.nomad.Jobs().Register(job, nil)
		if err == nil {
			logrus.Infof("rendering topology in evaluation id %s took %s", resp.EvalID, resp.RequestTime.String())
			deploymentFile.WriteString(fmt.Sprintf("%s\n", *job.ID))
			deploymentFile.Sync()
		} else {
			return err
		}
		if err = t.WaitEval(resp.EvalID); err != nil {
			return err
		}
		logrus.Infof("phase %d scheduled, running post deploy hooks...", i)
		for _, postDeployFunc := range postDeployFuncs[i] {
			if err := postDeployFunc(t.consul); err != nil {
				return err
			}
		}
		logrus.Infof("phase %d complete", i)
	}
	return nil
}
