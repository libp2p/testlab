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

func (n *Node) taskGroups(consul *capi.Client, baseName string, quantity int, options utils.NodeOptions) ([]*napi.TaskGroup, error) {
	bootstrappers := []string{}
	groups := make([]*napi.TaskGroup, quantity)

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
		task := napi.NewTask("ipfs", "docker")
		group := napi.NewTaskGroup(fmt.Sprintf("%s-%d", baseName, i), 1)

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
		group.AddTask(task)
		groups[i] = group
	}

	return groups, nil
}

func (n *Node) TaskGroups(consul *capi.Client, name string, quantity int, options utils.NodeOptions) ([]*napi.TaskGroup, error) {
	taskGroups, err := n.taskGroups(consul, name, quantity, options)
	if err != nil {
		return nil, err
	}

	return taskGroups, nil
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
