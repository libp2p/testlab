datacenter = "localvm"
data_dir = "/var/consul"
log_level = "INFO"
node_name = "testlab-dev-consul"
server = true
addresses {
    http = "0.0.0.0"
}
# bind_addr to be passed in command line options
bootstrap_expect = 1
ports {
    http = 8500
}
