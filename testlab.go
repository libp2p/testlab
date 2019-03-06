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

// TestLab is the main entrypoint for manipulating the test cluster.
type TestLab struct {
	path       string
	nomad      *napi.Client
	deployment string
}

// NewTestlab initiates a testlab, with a path to the current state of the
// testlab as well as a configuration for contacting the nomad cluster. If nil,
// nomadConfig will be populated with the defaults.
func NewTestlab(path string) (*TestLab, error) {
	nomadConfig := napi.DefaultConfig()

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
	job, err := topology.Job()
	if err != nil {
		return err
	}
	resp, _, err := t.nomad.Jobs().Register(job, wopts)
	if err == nil {
		logrus.Infof("rendering topology in evaluation id %s took %s", resp.EvalID, resp.RequestTime.String())
		err = ioutil.WriteFile(filepath.Join(t.path, "deployment"), []byte(topology.Name), 0644)
	}
	return err
}
