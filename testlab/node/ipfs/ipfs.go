package ipfs

import (
	"context"
	"fmt"
	"time"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	iapi "github.com/ipfs/go-ipfs-http-client"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/testlab/testlab/node/p2pd"
	"github.com/libp2p/testlab/testlab/node/prometheus"
	"github.com/libp2p/testlab/utils"
	"github.com/sirupsen/logrus"
)

const APIServiceName = "ipfsapi"
const GatewayServiceName = "ipfsgateway"

type Node struct{}

func (n *Node) tasks(consul *capi.Client, quantity int, options utils.NodeOptions) ([]*napi.Task, error) {
	bootstrappers := []string{}
	tasks := make([]*napi.Task, quantity)

	if bootstrap, ok := options.String("Bootstrap"); ok {
		addrs, err := utils.PeerControlAddrStrings(consul, p2pd.Libp2pServiceName, []string{bootstrap})
		if err != nil {
			return nil, err
		}
		ids, err := utils.PeerIDs(consul, addrs)
		if err != nil {
			return nil, err
		}
		for i, addr := range addrs {
			addrs[i] = fmt.Sprintf("%s/p2p/%s", addr, ids[i])
		}
		bootstrappers = addrs
	}

	for i := 0; i < quantity; i++ {
		task := napi.NewTask(fmt.Sprintf("ipfs-%d", i), "docker")

		version, ok := options.String("Version")
		if !ok {
			return nil, fmt.Errorf("ipfs plugin requires Version option to be set")
		}

		task.Config = map[string]interface{}{
			"image":        fmt.Sprintf("ipfs/go-ipfs:%s", version),
			"network_mode": "host",
		}

		res := napi.DefaultResources()
		res.Networks = []*napi.NetworkResource{
			{
				DynamicPorts: []napi.Port{
					{Label: p2pd.Libp2pServiceName},
					{Label: APIServiceName},
					{Label: GatewayServiceName},
					{Label: prometheus.MetricsServiceName},
				},
			},
		}
		task.Resources = res

		task.Services = []*napi.Service{
			{
				Name:        p2pd.Libp2pServiceName,
				PortLabel:   p2pd.Libp2pServiceName,
				AddressMode: "host",
			},
			{
				Name:        APIServiceName,
				PortLabel:   APIServiceName,
				AddressMode: "host",
				Checks: []napi.ServiceCheck{
					{
						Type:      "http",
						Path:      "/api/v0/id",
						Interval:  10 * time.Second,
						Timeout:   5 * time.Second,
						PortLabel: APIServiceName,
					},
				},
			},
			{
				Name:        GatewayServiceName,
				PortLabel:   GatewayServiceName,
				AddressMode: "host",
			},
		}

		if tags, ok := options.StringSlice("Tags"); ok {
			for _, service := range task.Services {
				service.Tags = tags
			}
		}

		ipfsRoot, ok := options.String("IpfsRootTemplate")
		if !ok {
			return nil, fmt.Errorf("ipfs plugin requires IpfsRootTemplate option set")
		}

		containerRoot := "/data/ipfs"

		task.Config["volumes"] = []interface{}{
			fmt.Sprintf("ipfs-config:%s", containerRoot),
		}

		if newRoot, ok := options.String("IpfsContainerRoot"); ok {
			containerRoot = newRoot
		}

		templates, err := ipfsConfiguration(ipfsRoot, "ipfs-config", bootstrappers)
		if err != nil {
			return nil, err
		}
		task.Templates = append(task.Templates, templates...)
		task.Env = map[string]string{
			"IPFS_LOGGING": "info",
		}
		tasks[i] = task
	}

	return tasks, nil
}

/*
 NOTES TO SELF:

for configuring these, we should really just make users provide a path that points to the
ipfs config directory, containing the minimum data required to start a new node. that way, all
we have to do is load the files into memory, do some variable substitution (let's use HCL) and
environment variable injection (to the Template struct)
*/

func (n *Node) TaskGroup(consul *capi.Client, name string, quantity int, options utils.NodeOptions) (*napi.TaskGroup, error) {
	group := napi.NewTaskGroup(name, 1)

	tasks, err := n.tasks(consul, quantity, options)
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		group.AddTask(task)
	}

	return group, nil
}

func (n *Node) PostDeploy(consul *capi.Client, options utils.NodeOptions) error {
	var (
		tags []string
		ok   bool
	)
	if tags, ok = options.StringSlice("Tags"); !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := utils.WaitService(ctx, consul, APIServiceName, tags)
	if err != nil {
		return err
	}
	addrs, err := utils.PeerControlAddrs(consul, APIServiceName, tags)
	if err != nil {
		return err
	}

	for _, addr := range addrs {
		client, err := iapi.NewApi(addr)
		if err != nil {
			logrus.Error(err)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		keyInfo, err := client.Key().Self(ctx)
		if err != nil {
			logrus.Error(err)
			continue
		}
		listenAddrs, err := client.Swarm().ListenAddrs(ctx)
		if err != nil {
			logrus.Error(err)
			continue
		}
		for _, listenAddr := range listenAddrs {
			if _, err := listenAddr.ValueForProtocol(circuit.P_CIRCUIT); err == nil {
				continue
			}
			err = utils.AddPeerIDToConsul(consul, keyInfo.ID().Pretty(), listenAddr.String())
			if err != nil {
				logrus.Error(err)
				continue
			}
		}
	}

	return nil
}
