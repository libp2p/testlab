package utils

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/go-libp2p-core/peer"
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
		Key:   fmt.Sprintf("peerids%s", addr),
		Value: []byte(peerID),
	}
	_, err := consul.KV().Put(kv, nil)
	return err
}

func WaitService(ctx context.Context, consul *capi.Client, service string, tags []string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			svcs, _, err := consul.Catalog().ServiceMultipleTags(service, tags, nil)
			if err != nil {
				return err
			}
			for _, svc := range svcs {
				if svc.Checks.AggregatedStatus() != capi.HealthPassing {
					continue
				}
			}
			return nil
		}
	}
}

func PeerControlAddrStrings(consul *capi.Client, service string, tags []string) ([]string, error) {
	svcs, _, err := consul.Catalog().ServiceMultipleTags(service, tags, nil)
	if err != nil {
		return nil, err
	}

	addrs := make([]string, len(svcs))
	for i, svc := range svcs {
		svc.Checks.AggregatedStatus()
		addrs[i] = fmt.Sprintf("/ip4/%s/tcp/%d", svc.ServiceAddress, svc.ServicePort)
	}

	return addrs, nil
}

func PeerControlAddrs(consul *capi.Client, service string, tags []string) ([]ma.Multiaddr, error) {
	addrs, err := PeerControlAddrStrings(consul, service, tags)
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

func PeerIDs(consul *capi.Client, addrs []string) ([]peer.ID, error) {
	ids := make([]peer.ID, len(addrs))
	for i, addr := range addrs {
		pair, meta, err := consul.KV().Get(fmt.Sprintf("peerids%s", addr), nil)
		if err != nil {
			return nil, err
		}
		if pair == nil {
			opts := &capi.QueryOptions{
				WaitIndex: meta.LastIndex,
				WaitTime:  10 * time.Second,
			}
			pair, meta, err = consul.KV().Get(addr, opts)
			if err != nil {
				return nil, err
			}
			if pair == nil {
				return nil, fmt.Errorf("key not found %s", addr)
			}
		}
		id, err := peer.IDB58Decode(string(pair.Value))
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}
