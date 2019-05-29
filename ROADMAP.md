# Testlab Roadmap

A rough checklist of what's to come for the testlab.

## Alpha Milestones

- Automation
  - [ ] Packer for images
    - [x] Local development
    - [ ] Cluster
  - [ ] Release default packer images on IPFS
  - [ ] Terraform for cluster provisioning
- Metrics collections
  - [x] Implement a wrapper for prometheus metrics collector that leverages
        consul to discover endpoints to scrape.
  - [x] Implement a plugin for prometheus metrics collector
- Scenario design
  - [ ] Finalize scenario execution environment spec (IN PROGRESS)
  - [x] Implement basic golang library that makes it easy to operate within this
        environment (e.g. automatically parses environment, makes it easy to
        create daemon clients)
  - [x] Implement example test scenario
  - [ ] Test in clustered environment
- Target Plugins
  - [x] Make existing p2pd plugin compatible with js implementation
  - [ ] Enable rudimentary provisioning for p2pd plugin (`npm install`, etc)
  - [x] IPFS plugin
