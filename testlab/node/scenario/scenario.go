package scenario

import (
	"fmt"
	"reflect"
	"strings"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/utils"
	"github.com/sirupsen/logrus"
)

type Node struct {
	consulConfig *capi.Config
}

func (s *Node) PostDeploy(consul *capi.Client, options utils.NodeOptions) error {
	return nil
}

func (s *Node) Task(options utils.NodeOptions) (*napi.Task, error) {
	task := napi.NewTask("scenario", "exec")
	task.Env = make(map[string]string)

	res := napi.DefaultResources()
	memory := 500
	res.MemoryMB = &memory
	cpu := 500
	res.CPU = &cpu

	if clients, ok := options.Int("Clients"); ok {
		task.Env["DAEMON_CLIENTS"] = fmt.Sprint(clients)
		dynamicPorts := make([]napi.Port, clients)
		for i := 0; i < clients; i++ {
			label := fmt.Sprintf("client%d", i)
			dynamicPorts[i] = napi.Port{Label: label}
		}
		res.Networks = append(res.Networks, &napi.NetworkResource{
			DynamicPorts: dynamicPorts,
		})
	} else {
		logrus.Fatal("missing required configuration option, Clients, in scenario")
	}

	task.Require(res)

	var command string
	if url, ok := options.String("Fetch"); ok {
		task.Artifacts = []*napi.TaskArtifact{
			&napi.TaskArtifact{
				GetterSource: utils.StringPtr(url),
				RelativeDest: utils.StringPtr("scenario"),
			},
		}
		command = "scenario"
	} else if cmd, ok := options.String("Command"); ok {
		command = cmd
	} else {
		return nil, fmt.Errorf(`scenarios require a "Fetch" or "Command" option be set, found neither`)
	}
	task.SetConfig("command", command)

	if serviceName, ok := options.String("TargetService"); ok {
		task.Env["SERVICE_NAME"] = serviceName
	} else {
		return nil, fmt.Errorf(`scenarios require a "TargetService" option be set, found none`)
	}

	if env, ok := options.Object("Env"); ok {
		for k, v := range env {
			vstr, ok := v.(string)
			if !ok {
				typ := reflect.TypeOf(v)
				logrus.Warnf("expected Env key %s to be a string, got %s", k, typ.String())
				continue
			}
			task.Env[k] = vstr
		}
	}

	if s.consulConfig != nil {
		envStrs := s.consulConfig.GenerateEnv()
		for _, envStr := range envStrs {
			parts := strings.SplitN(envStr, "=", 2)
			if len(parts) != 2 {
				continue
			}
			task.Env[parts[0]] = parts[1]
		}
	}

	return task, nil
}
