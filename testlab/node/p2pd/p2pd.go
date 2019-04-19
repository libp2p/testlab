package p2pd

import (
	"fmt"
	"path/filepath"

	"io/ioutil"
	"os"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	"github.com/libp2p/testlab/utils"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

type Node struct{}

func (n *Node) Task(options utils.NodeOptions) (*napi.Task, error) {
	task := napi.NewTask("p2pd", "exec")
	command := "/usr/local/bin/p2pd"
	args := []string{
		"-listen", "/ip4/${NOMAD_IP_p2pd}/tcp/${NOMAD_PORT_p2pd}",
		"-hostAddrs", "/ip4/${NOMAD_IP_libp2p}/tcp/${NOMAD_PORT_libp2p}",
		"-metricsAddr", "${NOMAD_ADDR_metrics}",
		"-pubsub", "-b",
	}

	if router, ok := options.String("PubsubRouter"); ok {
		args = append(args, "-pubsubRouter", router)
	}

	res := napi.DefaultResources()
	res.Networks = []*napi.NetworkResource{
		&napi.NetworkResource{
			DynamicPorts: []napi.Port{
				napi.Port{Label: "libp2p"},
				napi.Port{Label: "p2pd"},
				napi.Port{Label: "metrics"},
			},
		},
	}
	task.Require(res)

	p2pdSvc := &napi.Service{
		Name:        "p2pd",
		PortLabel:   "p2pd",
		AddressMode: "host",
	}
	metricsSvc := &napi.Service{
		Name:        "metrics",
		PortLabel:   "metrics",
		AddressMode: "host",
	}
	task.Services = append(task.Services, p2pdSvc, metricsSvc)

	url := ""

	if cid, ok := options.String("Cid"); ok {
		url = fmt.Sprintf("https://gateway.ipfs.io/ipfs/%s", cid)
	}

	if urlOpt, ok := options.String("Fetch"); ok {
		url = urlOpt
	}

	if url != "" {
		task.Artifacts = []*napi.TaskArtifact{
			&napi.TaskArtifact{
				GetterSource: utils.StringPtr(url),
				RelativeDest: utils.StringPtr("p2pd"),
			},
		}
		command = "p2pd"
	}

	// TODO: there should be a way to expose the libp2p service as well
	if service, ok := options.String("Service"); ok {
		if service == "p2pd" {
			logrus.Error("p2pd already exports service \"p2pd\"")
		} else {
			svc := &napi.Service{
				Name:        service,
				PortLabel:   "p2pd",
				AddressMode: "host",
			}
			task.Services = append(task.Services, svc)
		}
	}

	if bootstrap, ok := options["Bootstrap"]; ok {
		tmpl := `BOOTSTRAP_PEERS={{range $index, $service := service "%s"}}{{if ne $index 0}},{{end}}/ip4/{{$service.Address}}/tcp/{{$service.Port}}/p2p/{{printf "/peerids/%%s" $service.ID | key}}{{end}}`
		tmpl = fmt.Sprintf(tmpl, bootstrap)
		env := true
		template := &napi.Template{
			EmbeddedTmpl: &tmpl,
			DestPath:     utils.StringPtr("bootstrap_peers.env"),
			Envvars:      &env,
		}
		task.Templates = append(task.Templates, template)
		args = append(args, "-bootstrapPeers", "${BOOTSTRAP_PEERS}")
	}

	task.SetConfig("command", command)
	task.SetConfig("args", args)

	return task, nil
}

func (n *Node) PostDeploy(consul *capi.Client, options utils.NodeOptions) error {
	service, ok := options.String("Service")
	if !ok {
		logrus.Info("skipping post deploy for p2pd, no Service option")
		return nil
	}

	svcs, _, err := consul.Catalog().Service(service, "", nil)
	if err != nil {
		return err
	}
	bootstrapControlAddrs := make(map[string]ma.Multiaddr)
	for _, svc := range svcs {
		addrStr := fmt.Sprintf("/ip4/%s/tcp/%d", svc.ServiceAddress, svc.ServicePort)
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		bootstrapControlAddrs[svc.ServiceID] = addr
	}
	for svcID, addr := range bootstrapControlAddrs {
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
		peerID, _, err := client.Identify()
		if err != nil {
			return err
		}
		kv := &capi.KVPair{
			Key:   fmt.Sprintf("peerids/%s", svcID),
			Value: []byte(peerID.Pretty()),
		}
		_, err = consul.KV().Put(kv, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
