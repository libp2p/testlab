#!/bin/bash

CONSUL_ZIP=/tmp/nomad.zip

if [[ "$?" -ne "0" ]]; then
  echo "failed to install unzip";
  exit 1
fi

curl -o $CONSUL_ZIP https://releases.hashicorp.com/consul/1.4.4/consul_1.4.4_linux_amd64.zip

if [[ "$?" -ne "0" || ! -e $CONSUL_ZIP ]]; then
  echo "failed to download nomad";
  exit 1
fi

cd /usr/local/bin &&
  unzip $CONSUL_ZIP

if [[ "$?" -ne "0" ]]; then
  echo "failed to extract nomad";
  exit 1
fi

chmod 0755 /usr/local/bin/consul &&
  chown root:root /usr/local/bin/consul &&
  mkdir /var/consul

# cleanup
rm $CONSUL_ZIP

