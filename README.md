# Testlab

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://protocol.ai)
[![](https://img.shields.io/badge/project-libp2p-blue.svg?style=flat-square)](http://libp2p.io/)
[![](https://img.shields.io/badge/freenode-%23libp2p-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%libp2p)
[![GoDoc](https://godoc.org/github.com/libp2p/testlab?status.svg)](https://godoc.org/github.com/libp2p/testlab)

> A cluster-ready testlab, suitable for monitoring the behavior of p2p systems
  at scale. Built on nomad and consul.

🚧 This project is under active development! 🚧

Check out the [ROADMAP](ROADMAP.md) to see what's coming.

## Table of Contents

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Dependencies](#dependencies)
  - [Development Cluster](#development-cluster)
  - [Notes](#notes)
  - [Production Cluster](#production-cluster)
- [Installation](#installation)
- [How it Works](#how-it-works)
- [Usage](#usage)
  - [CLI](#cli)
  - [Deployment Configuration](#deployment-configuration)
    - [`Name: string`](#name-string)
    - [`Options: object`](#options-object)
    - [`Deployments: list of objects`](#deployments-list-of-objects)
      - [`Name: string`](#name-string-1)
      - [`Plugin: string`](#plugin-string)
      - [`Quantity: int`](#quantity-int)
      - [`Options: object`](#options-object-1)
      - [`Dependencies: list`](#dependencies-list)
  - [Scenario Runners](#scenario-runners)
  - [Node API](#node-api)
  - [Node Implementations](#node-implementations)
    - [p2pd](#p2pd)
      - [Options](#options)
      - [Post Deploy Hook](#post-deploy-hook)
    - [scenario](#scenario)
      - [Options](#options-1)
      - [Post Deploy Hook](#post-deploy-hook-1)
    - [prometheus](#prometheus)
      - [Options](#options-2)
      - [Post Deploy Hook](#post-deploy-hook-2)
- [Contribute](#contribute)
- [Help Wanted](#help-wanted)
- [License](#license)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Dependencies

- [Nomad](https://nomadproject.io)
- [Consul](https://consul.io)
- [Packer](https://packer.io)
- [Terraform](https://www.terraform.io/)

You'll need a nomad cluster which, in turn, requires a consul deployment, in
order to run testlab.

### Development Cluster

In development, the configuration files in
`automation/packer/config` should be sufficient to run a single node
deployment. Furthermore, the packer configuration, with the help of the
`Makefile` can build a simple VM image for either VMWare or VirtualBox,
featuring a testlab binary. Try the commands:

```
$ make vm-virtualbox
```

or

```
make vm-vmware
```

### Notes

When deploying nomad manually, you must take care to deploy the nomad
agent as root, since it requires on cgroups and/or docker to launch sandboxed
tasks.

### Production Cluster

In production, a larger nomad deployment is advised. Hashicorp has recipes for
[deploying nomad clusters on aws](https://github.com/hashicorp/terraform-aws-nomad).
In the near future, testlab will include its own terraform recipes in the
`automation` directory.

## Installation

Testlab is a simple go binary, and can be installed into your GOPATH as such:

```
$ go get github.com/libp2p/testlab/testlab
```

## How it Works

Testlab is an automation layer over Hashicorp's
[Nomad](https://nomadproject.io), a cluster manager in the same style as
Kubernetes. Testlab's primary goal is to make it simple to launch large clusters
of peer-to-peer applications to better understand how they function at scale.

Testlab topologies are built around two main concepts: peer deployments and
scenario runners. Generally, a peer deployment describes a set of instances of a
peer-to-peer application and, optionally, how they are connected. A scenario
runner is a special program launched in the cluster that can remotely control
peer deployments to simulate activity within the network.

The goal output of a testlab topology is metrics data. While, in the future,
it would be nice to support correctness tests, the current aim is to allow for
large scale benchmarking and diagnosis of issues, as well as regression
testing. All peer deployments should be instrumented with prometheus-friendly
metrics, should they want to have data collected. This is described in greater
detail in the [scenario runners](#scenario-runners) section. Clusters
specifying a deployment of the `prometheus` plugin will automatically have
metrics collected.

## Usage

Testlab is a wrapper over [nomad](https://www.nomadproject.io/)'s golang API,
making it easy to deploy pre-configured networks of p2p applications.

Most users of testlab need only concern themselves with two concepts, the
[deployment configuration](#deployment-configuration), and
[scenarios](#scenario-runners). Users wishing to add testlab support for their
own daemons will need to understand the [node API](#node-api), as well.

### CLI

The testlab CLI depends on the presence of the standard environment variables to
connect out to your Nomad and Consul clusters. If any are ommitted, the
defaults, as defined by Hashicorp, will be applied. The defaults are typically
usable in development.

- [Nomad environment variables](https://github.com/hashicorp/nomad/blob/v0.9.3/api/api.go#L217)
- [Consul environment variables](https://github.com/hashicorp/consul/blob/v1.5.2/api/api.go#L24)

Furthermore, users can optionally provide a path in the environment variable
`TESTLAB_ROOT` to define where the testlab metadata will be stored. This
defaults to `/tmp/testlab`. **NOTE**: In order to have multiple testlab
topologies in flight at the same time, one must define different `TESTLAB_ROOT`s
for each topology. This requirement exists as a result of testlab associating a
single nomad deployment ID with each `TESTLAB_ROOT`, though this can be extended
quite easily in the future.

The testlab CLI has two commands:

- `testlab start <json configuration>`
  Parses, evaluates for correctness, and attempts to deploy a topology as
  defined by the provided json configuration file. Once all of the peer-to-peer
  nodes a scenario depends on are deployed, the scenario will be deployed.
- `testlab stop`
  Stops the current running topology, identified by its `TESTLAB_ROOT`.

### Deployment Configuration

The entrypoint for most projects using the testlab will be their deployment
configuration, a JSON document declaring the desired network configuration.
It's broken into the following top level sections:

#### `Name: string`

The name of the deployment. This will become a prefix to all tasks launched
in the testlab.

#### `Options: object`

Cluster-wide options to apply to the deployment.

```
{
    // Datacenters is a list of nomad datacenters on which this test deployment
    // should be scheduled. Nomad supports multiple datacenter deployments. By
    // default this should be all datacenters.
    "Datacenters": list of strings,

    // Priority is an integer from [1, 100], the higher the more important. This
    // allows nomad to determine which tasks should be scheduled when there is
    // resource contention. If your nomad cluster has other tasks running on it,
    // be sure to set this value accordingly. Otherwise, a default of 50 will be
    // provided.
    "Priority": int,
}
```

#### `Deployments: list of objects`

The deployments are where it gets interesting! Each deployment defines a class
of node to be scheduled on the cluster. Each deployment **must** define a
**`Name`**, **`Plugin`**, and **`Quantity`** and may optionally define
**`Options`** specific to the plugin and **`Dependencies`**.

##### `Name: string`

The name of this set of peers. This name will be used to reference these peers
in the **`Dependencies`**.

##### `Plugin: string`

Defines which node plugin to use. This defines how these nomad tasks will be
configured. Must be one of the string identifiers listed in the
[node implementations](#node-implementations) section.

##### `Quantity: int`

Defines how many of this type of peer should be launched in the cluster.

##### `Options: object`

An optional object as defined by the specific
[node implementation](#node-implementations).

##### `Dependencies: list`

A list of **`Name`** s of deployments that must be scheduled before this one.
This feature exists for many reasons, such as allowing gateway nodes to go up
before generic peers that might want to bootstrap on them, or ensuring a
deployment of peers is launched before. The scenario that drives them is
scheduled. Cycles are not permitted.

### Scenario Runners

Scenario runners are the beating heart of testlab's simulation capabilities.
It is their responsibility to drive the various deployments to create activity
within the network. While it's not entirely necessary to use the `scenario` node
to deploy a scenario runner, it can be quite useful, especially in larger
clusters.

The scenario runner API is described by its
[node implementation](testlab/node/scenario/scenario.go) **and is, at present, a
work in progress**. [Pull requests welcome!](#help-wanted)

Scenario runners can expect a few environment variables to be present, to aid
them in connecting to the peers they wish to control. These variables are mostly
tailored towards helping them interact with Consul, to discover information
about the peers they've been assigned to.

- `DAEMON_CLIENTS` (int): The number of TCP/UDP ports this scenario runner has
  been allocated. These ports can be used for callbacks from daemons, such as
  how the libp2p daemon uses callbacks to receive incoming streams, etc.
  **TODO**: This should be become a more generic key, like`TESTLAB_PORTS`.
- `SERVICE_TAG` (string): The tag that will be applied to the Consul services
  this runner is meant to control. For example, if a scenario is controlling
  libp2p daemons, which expose a `p2pd` service for daemon control, it could
  query the consul cluster for `p2pd` services with the `$SERVICE_TAG` tag,
  yielding the daemon control port of every daemon under their purview.
- `CONSUL_*` (various): Additionally, the standard set of
  [consul environment variables](https://www.consul.io/docs/commands/index.html#environment-variables)
  will be present, so that the scenario may connect to the consul cluster.

As will be documented below in the [node implementations](#node-implementations)
section, users can pass in any additional environment variables they wish to
their scenario runner via the `Env` option in their configuration.

This set of environment variables is the extent of the scenario runner "API". It
is up to the user how to use these. If working in golang, one can use the
[nascent golang scenario runner API](scenario/scenario.go), which provides
convenience functions for accessing consul and creating libp2p daemon clients.
**TODO**: Generalize this library to focus entirely on consul access, and split
libp2p specific functionality into a separate sub-package.

### Node API

Nodes describe how peer-to-peer applications should be launched within the
cluster. In order to add testlab support for your peer-to-peer application, you
must implement the following api

```go
package node

import (
	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	utils "github.com/libp2p/testlab/utils"
)

type Node interface {
	Task(utils.NodeOptions) (*napi.Task, error)
	PostDeploy(*capi.Client, utils.NodeOptions) error
}
```

Given some `utils.NodeOptions`, a wrapper over the `map[string]interface{}` type
generated by JSON deserialization in go, a `Node` must generate a
[Nomad task](https://www.nomadproject.io/docs/job-specification/task.html) or
return an error.

Furthermore, a `Node` must implement a post-deployment hook (can be no-op), a
function that is called after deployments of this type have been successfully
scheduled in the cluster. This can be useful for connecting to the newly
launched peers and writing important metadata pertaining to them into Consul's
KV store. An example of this is the libp2p daemon, which uses it to associate
a peer's randomly generated ID with it's consul service ID.

### Node Implementations

At present, there are three node implementations:

- `p2pd`: the libp2p daemon
- `scenario`: the generic scenario runner
- `prometheus`: prometheus metrics collection

A description of their behavior and configuration options follows.

#### p2pd

The p2pd plugin adds support for the
[libp2p daemon](https://github.com/libp2p/go-libp2p-daemon). It will spawn
libp2p peers, exposing the following services:

- `libp2p`: The libp2p host.
- `p2pd`: The libp2p daemon control endpoint, exposed so scenario runners can
  manipulate the peer.
- `metrics`: Prometheus scraping endpoint.

##### Options

libp2p daemons can be configured with the following options:

- `PubsubRouter` string (optional): "gossipsub" or "floodsub", per users preference.
- `Cid` string (optional): instead of looking for the `p2pd` binary on the local
  filesystem, testlab can fetch a binary from IPFS by it's Cid.
- `Fetch` string (optional): instead of looking for the `p2pd` binary on the local
  filesystem, testlab can fetch a binary from an arbitrary (http/s) URL.
- `Tags` list of strings (optional): Tags to apply to the service entries in
  Consul. These make it possible for scenarios to reference the specific subset
  of peers they're assigned to manipulate.
- `Bootstrap` string (optional): The name of another deployment representing
  the network's "bootstrapper" (well known entrypoint) nodes. These will be
  automatically connected to when the daemon starts.

##### Post Deploy Hook

After the libp2p daemons are successfully scheduled on the cluster, testlab will
query each peer for its peer ID and store it in the Consul KV store under the
key `"peerid/<multiaddr to libp2p service>"` e.g. `peerid/ip4/127.0.0.1/tcp/6`.

#### scenario

The scenario plugin adds support for launching scenario runners in the testlab
cluster. They must either be present on the clusters /usr/... path, or can be
fetched from a URL like the libp2p daemon. Scenario runners will be provided
environment variables as described above. 

##### Options

Scenario runners can be configured with the following options:

- `Clients` int (required): The number of TCP/UDP ports to allocate for this
  scenario. So-named because the libp2p daemon requires ports in order to
  receive information pushed from the daemon. **TODO**: Generalize this.
- `Fetch` string (optional): instead of looking for the `p2pd` binary on the
  local filesystem, testlab can fetch a binary from an arbitrary (http/s) URL.

##### Post Deploy Hook

None.

#### prometheus

The prometheus plugin adds support for launching a
[Prometheus](https://prometheus.io/) metrics collector. Testlab automatically
configures prometheus to scrape Consul for all tasks exposing a `metrics`
service.

**NOTE**: As previously mentioned, all `CONSUL_*` and `NOMAD_*` environment
variables must be defined in the terminal that `testlab` is executed from. If
they are not, they will not be passed along to the prometheus configuration.
This can result in prometheus failing to scrape Consul.

**NOTE**: Currently, a prometheus node still needs to be manually added to the
topology configuration. This may become automatic in the future.

##### Options

None.

##### Post Deploy Hook

None.

## Contribute

Feel free to join in. All welcome. Open an [issue](https://github.com/libp2p/testlab/issues)!

This repository falls under the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

## Help Wanted

If you've got a peer-to-peer application you'd like to start testing and
benchmarking at scale, don't hesitate to submit a PR adding a `Node` for it!
Please feel free to ask any questions in the issues or on #libp2p on freenode.

## License

MIT / Apache 2 Dual License
