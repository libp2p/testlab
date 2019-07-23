package ipfs

import (
	"fmt"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/testlab/node/p2pd"
	"github.com/libp2p/testlab/testlab/node/prometheus"
	"github.com/libp2p/testlab/utils"
)

const APIServiceName = "ipfsapi"
const GatewayServiceName = "ipfsgateway"

type Node struct{}

/*
 NOTES TO SELF:

for configuring these, we should really just make users provide a path that points to the
ipfs config directory, containing the minimum data required to start a new node. that way, all
we have to do is load the files into memory, do some variable substitution (let's use HCL) and
environment variable injection (to the Template struct)
*/

func (n *Node) Task(consul *capi.Client, options utils.NodeOptions) (*napi.Task, error) {
	task := napi.NewTask("ipfs", "docker")

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
	bootstrappers := []string{}

	if bootstrap, ok := options.String("Bootstrap"); ok {
		addrs, err := utils.PeerControlAddrStrings(consul, p2pd.Libp2pServiceName, bootstrap)
		if err != nil {
			return nil, err
		}
		bootstrappers = addrs
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

	return task, nil
}

func (n *Node) PostDeploy(consul *capi.Client, options utils.NodeOptions) error {
	//if bootstrapTag, ok := options.String("Bootstrap"); ok {
	//}
	return nil
}
