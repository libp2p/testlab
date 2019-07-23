package utils

import (
	"fmt"
	"os"
	"regexp"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	ma "github.com/multiformats/go-multiaddr"
)

// StringPtr stack allocates a string literal and returns a reference to it
func StringPtr(str string) *string {
	return &str
}

// ValidTaskNameRegexp matches task names consisting of alpha-numeric characters
// and dashes
var ValidTaskNameRegexp = regexp.MustCompile(`(?i)^[A-Za-z0-9\-]+$`)

var consulEnvNames = []string{
	capi.GRPCAddrEnvName,
	capi.HTTPAddrEnvName,
	capi.HTTPAuthEnvName,
	capi.HTTPSSLEnvName,
	capi.HTTPSSLVerifyEnvName,
	capi.HTTPTokenEnvName,
	capi.HTTPCAFile,
	capi.HTTPCAPath,
	capi.HTTPClientCert,
	capi.HTTPClientKey,
	capi.HTTPTLSServerName,
}

// AddConsulEnvToTask inspects the environment for consul-related variables,
// adding any found to the given task's environment. TODO: Add defaults support
func AddConsulEnvToTask(t *napi.Task) {
	for _, envVar := range consulEnvNames {
		if envVal, ok := os.LookupEnv(envVar); ok {
			t.Env[envVar] = envVal
		}
	}
}

func AddPeerIDToConsul(consul *capi.Client, peerID string, addr string) error {
	kv := &capi.KVPair{
		Key: fmt.Sprintf("peerids%s", addr),
		Value: []byte(peerID),
	}
	_, err := consul.KV().Put(kv, nil)
	return err
}

func PeerControlAddrStrings(consul *capi.Client, service, tag string) ([]string, error) {
	svcs, _, err := consul.Catalog().Service(service, tag, nil)
	if err != nil {
		return nil, err
	}

	addrs := make([]string, len(svcs))
	for i, svc := range svcs {
		addrs[i] = fmt.Sprintf("/ip4/%s/tcp/%d", svc.ServiceAddress, svc.ServicePort)
	}

	return addrs, nil
}

func PeerControlAddrs(consul *capi.Client, service, tag string) ([]ma.Multiaddr, error) {
	addrs, err := PeerControlAddrStrings(consul, service, tag)
	if err != nil {
		return nil, err
	}
	maddrs := make([]ma.Multiaddr, len(addrs))
	for i, addr := range addrs {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		maddrs[i] = maddr
	}

	return maddrs, nil
}
