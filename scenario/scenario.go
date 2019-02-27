package scenario

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	capi "github.com/hashicorp/consul/api"
	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
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
	service      string
}

func NewScenarioRunner() (*ScenarioRunner, error) {
	consulConfig := capi.DefaultConfig()
	// TODO: make a fn for testing that allows users to pass this in
	root, err := ioutil.TempDir(os.TempDir(), "scenario")
	if err != nil {
		return nil, err
	}
	var service string
	var ok bool
	service, ok = os.LookupEnv("SERVICE_NAME")
	if !ok {
		return nil, fmt.Errorf("SERVICE_NAME not present in environment")
	}

	runner := &ScenarioRunner{
		consulConfig: consulConfig,
		root:         root,
		service:      service,
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

	svcs, _, err := client.Catalog().Service(s.service, "", nil)
	if err != nil {
		return nil, err
	}

	maddrs := make([]ma.Multiaddr, len(svcs))
	for i, svc := range svcs {
		addr := fmt.Sprintf("/ip4/%s/tcp/%s", svc.ServiceAddress, svc.ServicePort)
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		maddrs[i] = maddr
	}

	return maddrs, nil
}

func (s *ScenarioRunner) Peers() ([]*p2pclient.Client, error) {
	path := filepath.Join(s.root, "clients")
	err := os.MkdirAll(path, 0777)
	if err != nil {
		return nil, fmt.Errorf("making clients directory: %s", err)
	}

	addrs, err := s.PeerControlAddrs()
	if err != nil {
		return nil, err
	}

	clientch := make(chan *p2pclient.Client)
	defer close(clientch)
	errch := make(chan error)
	defer close(errch)
	var wg sync.WaitGroup

	wg.Add(len(addrs))
	for i, addr := range addrs {
		go func() {
			defer wg.Done()
			listenSock := fmt.Sprintf("/unix/%s/clients/%d.sock", s.root, i)
			listenMaddr, err := ma.NewMultiaddr(listenSock)
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
		}()
	}
	wg.Done()

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
