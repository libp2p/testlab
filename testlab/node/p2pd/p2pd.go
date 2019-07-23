package p2pd

import (
	"fmt"
	"path/filepath"
	"strings"
	"io/ioutil"
	"os"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	"github.com/libp2p/testlab/utils"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

const Libp2pServiceName = "libp2p"
const DaemonServiceName = "p2pd"

type Node struct{}

func (n *Node) Task(consul *capi.Client, options utils.NodeOptions) (*napi.Task, error) {
	task := napi.NewTask("p2pd", "exec")
	command := "/usr/local/bin/p2pd"
	args := []string{
		"-listen", "/ip4/${NOMAD_IP_p2pd}/tcp/${NOMAD_PORT_p2pd}",
		"-metricsAddr", "${NOMAD_ADDR_metrics}",
		"-pubsub",
	}

	if router, ok := options.String("PubsubRouter"); ok {
		args = append(args, "-pubsubRouter", router)
	}

	res := napi.DefaultResources()
	res.Networks = []*napi.NetworkResource{
		&napi.NetworkResource{
			DynamicPorts: []napi.Port{
				napi.Port{Label: Libp2pServiceName},
				napi.Port{Label: DaemonServiceName},
				napi.Port{Label: "metrics"},
			},
		},
	}
	task.Require(res)

	metricsSvc := &napi.Service{
		Name:        "metrics",
		PortLabel:   "metrics",
		AddressMode: "host",
	}
	p2pdSvc := &napi.Service{
		Name:        DaemonServiceName,
		PortLabel:   DaemonServiceName,
		AddressMode: "host",
	}
	task.Services = append(task.Services, metricsSvc, p2pdSvc)

	if noBind, ok := options.Bool("Undialable"); !ok || !noBind {
		args = append(args, "-hostAddrs", "/ip4/${NOMAD_IP_libp2p}/tcp/${NOMAD_PORT_libp2p}")
		libp2pSvc := &napi.Service{
			Name:        Libp2pServiceName,
			PortLabel:   Libp2pServiceName,
			AddressMode: "host",
		}
		task.Services = append(task.Services, libp2pSvc)
	} else {
		args = append(args, "-noListenAddrs")
	}

	url := ""

	if cid, ok := options.String("Cid"); ok {
		url = fmt.Sprintf("https://gateway.ipfs.io/ipfs/%s", cid)
	}

	if urlOpt, ok := options.String("Fetch"); ok {
		url = urlOpt
	}

	if url != "" {
		task.Artifacts = []*napi.TaskArtifact{
			{
				GetterSource: utils.StringPtr(url),
				RelativeDest: utils.StringPtr("p2pd"),
			},
		}
		command = "p2pd"
	}

	if tags, ok := options.StringSlice("Tags"); ok {
		for _, service := range task.Services {
			service.Tags = tags
		}
	}

	if bootstrap, ok := options.String("Bootstrap"); ok {
		addrs, err := utils.PeerControlAddrStrings(consul, Libp2pServiceName, bootstrap)
		if err != nil {
			return nil, err
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("expected at least one %s.libp2p service, found none", bootstrap)
		}
		tmpl := fmt.Sprintf("BOOTSTRAP_PEERS=%s", strings.Join(addrs, ","))
		env := true
		template := &napi.Template{
			EmbeddedTmpl: &tmpl,
			DestPath:     utils.StringPtr("bootstrap_peers.env"),
			Envvars:      &env,
		}
		task.Templates = append(task.Templates, template)
		args = append(args, "-b", "-bootstrapPeers", "${BOOTSTRAP_PEERS}")
	}

	task.SetConfig("command", command)
	task.SetConfig("args", args)

	return task, nil
}

func (n *Node) PostDeploy(consul *capi.Client, options utils.NodeOptions) error {
	tags, ok := options.StringSlice("Tags")
	if !ok {
		logrus.Info("skipping post deploy for p2pd, no Tags option")
		return nil
	}

	svcs, _, err := consul.Catalog().ServiceMultipleTags(DaemonServiceName, tags, nil)
	if err != nil {
		return err
	}
	bootstrapControlAddrs := make([]ma.Multiaddr, len(svcs))
	for i, svc := range svcs {
		addrStr := fmt.Sprintf("/ip4/%s/tcp/%d", svc.ServiceAddress, svc.ServicePort)
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		bootstrapControlAddrs[i] = addr
	}
	for _, addr := range bootstrapControlAddrs {
		dir, err := ioutil.TempDir(os.TempDir(), "daemon_client")
		if err != nil {
			return err
		}
		sockPath := filepath.Join("/unix", dir, "ignore.sock")
		listenAddr, _ := ma.NewMultiaddr(sockPath)
		client, err := p2pclient.NewClient(addr, listenAddr)
		if err != nil {
			return err
		}
		defer func() {
			client.Close()
			os.RemoveAll(dir)
		}()
		peerID, addrs, err := client.Identify()
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			err = utils.AddPeerIDToConsul(consul, peerID.Pretty(), addr.String())
			if err != nil {
				return err
			}
		}
	}
	return nil
}
