#!/bin/bash

NOMAD_ZIP=/tmp/nomad.zip

if [[ "$?" -ne "0" ]]; then
  echo "failed to install unzip";
  exit 1
fi

curl -o $NOMAD_ZIP https://releases.hashicorp.com/nomad/0.8.7/nomad_0.8.7_linux_amd64.zip;

if [[ "$?" -ne "0" || ! -e $NOMAD_ZIP ]]; then
  echo "failed to download nomad";
  exit 1
fi

cd /usr/local/bin &&
  unzip $NOMAD_ZIP


if [[ "$?" -ne "0" ]]; then
  echo "failed to extract nomad";
  exit 1
fi

chmod 0755 /usr/local/bin/nomad &&
  chown root:root /usr/local/bin/nomad &&
  mkdir /var/nomad

# cleanup
rm $NOMAD_ZIP

