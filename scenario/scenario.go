package scenario

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"

	capi "github.com/hashicorp/consul/api"
	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	"github.com/libp2p/testlab/utils"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// Scenarios are run in an environment with the following information:
// - SERVICE_NAME

type Scenario struct {
	Name string
	Run  func(*ScenarioRunner) error
}

func RunScenario(scenario Scenario) {
	logrus.Infof("running scenario %s", scenario.Name)
}

type ScenarioRunner struct {
	consulConfig *capi.Config
	consul       *capi.Client
	root         string
	tag          string
	numClients   int
}

func NewScenarioRunner() (*ScenarioRunner, error) {
	consulConfig := capi.DefaultConfig()
	// TODO: make a fn for testing that allows users to pass this in
	root, err := ioutil.TempDir(os.TempDir(), "scenario")
	if err != nil {
		return nil, err
	}
	var tag string
	var ok bool
	tag, ok = os.LookupEnv("SERVICE_TAG")
	if !ok {
		return nil, fmt.Errorf("SERVICE_TAG not present in environment")
	}
	var numClients int
	if numClientsStr, ok := os.LookupEnv("DAEMON_CLIENTS"); ok {
		numClients, err = strconv.Atoi(numClientsStr)
		if err != nil {
			return nil, fmt.Errorf("expected DAEMON_CLIENTS to be an integer, found: %s", numClientsStr)
		}
	} else {
		return nil, fmt.Errorf("DAEMON_CLIENTS not present in environment")
	}

	runner := &ScenarioRunner{
		consulConfig: consulConfig,
		root:         root,
		tag:          tag,
		numClients:   numClients,
	}

	return runner, nil
}

// NewConsulClient creates a consul client from the given environment.
func (s *ScenarioRunner) ConsulClient() (*capi.Client, error) {
	if s.consul != nil {
		return s.consul, nil
	}

	client, err := capi.NewClient(s.consulConfig)
	if err != nil {
		return nil, err
	}

	s.consul = client
	return s.consul, nil
}

func (s *ScenarioRunner) PeerControlAddrs() ([]ma.Multiaddr, error) {
	client, err := s.ConsulClient()
	if err != nil {
		return nil, err
	}

	return utils.PeerControlAddrs(client, "p2pd", s.tag)
}

func (s *ScenarioRunner) Peers() ([]*p2pclient.Client, error) {
	addrs, err := s.PeerControlAddrs()
	if err != nil {
		return nil, err
	}

	clientch := make(chan *p2pclient.Client, 10)
	errch := make(chan error, 10)
	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(5)

	wg.Add(len(addrs))
	for i, addr := range addrs {
		if i > s.numClients {
			wg.Done()
			logrus.Warnf("skipping client creation for %s, already exceeded allocated ports", addr.String())
			continue
		}
		go func(i int, addr ma.Multiaddr, wg *sync.WaitGroup) {
			sem.Acquire(context.Background(), 1)
			defer sem.Release(1)
			defer wg.Done()

			clientHostVar := fmt.Sprintf("NOMAD_IP_client%d", i)
			clientPortVar := fmt.Sprintf("NOMAD_PORT_client%d", i)
			clientHost, ok := os.LookupEnv(clientHostVar)
			if !ok {
				errch <- fmt.Errorf("%s was not found in environment", clientHostVar)
				return
			}
			clientPort, ok := os.LookupEnv(clientPortVar)
			if !ok {
				errch <- fmt.Errorf("%s was not found in environment", clientPortVar)
				return
			}
			listenAddr := fmt.Sprintf("/ip4/%s/tcp/%s", clientHost, clientPort)
			listenMaddr, err := ma.NewMultiaddr(listenAddr)
			if err != nil {
				errch <- fmt.Errorf("creating control socket multiaddr: %s", err)
				return
			}
			client, err := p2pclient.NewClient(addr, listenMaddr)
			if err != nil {
				errch <- err
				return
			}
			clientch <- client
		}(i, addr, &wg)
	}
	wg.Wait()
	close(clientch)
	close(errch)

	// Should errors be fatal, or should we just log?
	if err, ok := <-errch; ok {
		return nil, err
	}

	clients := make([]*p2pclient.Client, 0, len(addrs))
	for client := range clientch {
		clients = append(clients, client)
	}

	return clients, nil
}
