package utils

import (
	"os"
	"regexp"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
)

func StringPtr(str string) *string {
	return &str
}

var ValidTaskNameRegexp = regexp.MustCompile(`(?i)^[A-Za-z0-9\-]+$`)

var consulEnvNames = []string{
	capi.GRPCAddrEnvName,
	capi.HTTPAddrEnvName,
	capi.HTTPAuthEnvName,
	capi.HTTPSSLVerifyEnvName,
	capi.HTTPTokenEnvName,
}

func AddConsulEnvToTask(t *napi.Task) {
	for _, envVar := range consulEnvNames {
		if envVal, ok := os.LookupEnv(envVar); ok {
			t.Env[envVar] = envVal
		}
	}
}
