data_dir  = "/var/nomad"

# bind_addr = "192.168.1.3" # the default

server {
  enabled          = true
  bootstrap_expect = 1
}

client {
  enabled       = true
  options {
    "driver.raw_exec.enable" = "1"
  }
}

consul {
  address = "127.0.0.1:8500"
}

