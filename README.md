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

- [Dependencies](#dependencies)
- [Installation](#installation)
- [How it Works](#how-it-works)
- [Usage](#usage)
- [Contribute](#contribute)
- [License](#license)

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
scenario runners. Generally, a peer deployment is set of instances of a
peer-to-peer application, and a scenario runner is a special program launched
in the cluster that can remotely control peer deployments to simulate activity
within the network.

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

A list of **`Name`**s of deployments that must be scheduled before this one.
This feature exists for many reasons, such as allowing gateway nodes to go up
before generic peers that might want to bootstrap on them, or ensuring a
deployment of peers is launched before. The scenario that drives them is
scheduled. Cycles are not permitted.

### Scenario Runners

### Node API

### Node Implementations

## Contribute

Feel free to join in. All welcome. Open an [issue](https://github.com/libp2p/testlab/issues)!

This repository falls under the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

## License

MIT / Apache 2 Dual License
