build:
	go build -o build/testlab ./testlab

vm-binary:
	GOOS="linux" go build -o build/testlab-vm ./testlab

vm: vm-binary
	PACKER_CACHE_DIR=./automation/packer/packer_cache packer build automation/packer/testlab-dev.json

vm-vmware: vm-binary
	PACKER_CACHE_DIR=./automation/packer/packer_cache packer build -only=vmware-iso automation/packer/testlab-dev.json
