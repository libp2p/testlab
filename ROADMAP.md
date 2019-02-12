# Testlab Roadmap

- Automation
  - [ ] Packer for images
  - [ ] Release default packer images on IPFS
  - [ ] Terraform for cluster provisioning
- Metrics collections
  - [ ] Implement a wrapper for prometheus metrics collector that leverages
        consul to discover endpoints to scrape.
  - [ ] Implement a plugin for prometheus metrics collector
- Scenario design
  - [ ] Finalize scenario execution environment spec
  - [ ] Implement basic golang library that makes it easy to operate within this
        environment (e.g. automatically parses environment, makes it easy to
        create daemon clients)
  - [ ] Implement example test scenario
- Target Plugins
  - [ ] Make existing p2pd plugin compatible with js implementation (this will
        likely require basic provisioning capabilities e.g. `npm install`)
  - [ ] IPFS plugin
