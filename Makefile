build: $(shell find . -type f -name '*.go')
	mkdir -p build
	go build -o build/testlab ./testlab

clean:
	rm -r build

vm-binary: $(shell find . -type f -name '*.go')
	GOOS="linux" go build -o build/testlab-vm ./testlab

vm: vm-binary $(shell find automation/packer -type f)
	PACKER_CACHE_DIR=./automation/packer/packer_cache packer build automation/packer/testlab-dev.json

vm-virtualbox: vm-binary $(shell find automation/packer -type f)
	PACKER_CACHE_DIR=./automation/packer/packer_cache packer build -only=virtualbox-iso automation/packer/testlab-dev.json

vm-vmware: vm-binary $(shell find automation/packer -type f)
	PACKER_CACHE_DIR=./automation/packer/packer_cache packer build -only=vmware-iso automation/packer/testlab-dev.json
