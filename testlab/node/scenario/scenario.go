package scenario

import (
	"fmt"
	"strings"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/utils"
)

type ScenarioNode struct {
	consulConfig *capi.Config
}

func (s *ScenarioNode) Task(options map[string]string) (*napi.Task, error) {
	task := napi.NewTask("scenario", "exec")

	res := napi.DefaultResources()
	task.Require(res)

	var command string
	if url, ok := options["Fetch"]; ok {
		task.Artifacts = []*napi.TaskArtifact{
			&napi.TaskArtifact{
				GetterSource: utils.StringPtr(url),
				RelativeDest: utils.StringPtr("scenario"),
			},
		}
		command = "scenario"
	} else if cmd, ok := options["Command"]; ok {
		command = cmd
	} else {
		return nil, fmt.Errorf(`scenarios require a "Fetch" or "Command" option be set, found neither`)
	}
	task.SetConfig("command", command)

	if serviceName, ok := options["TargetService"]; ok {
		task.Env["SERVICE_NAME"] = serviceName
	} else {
		return nil, fmt.Errorf(`scenarios require a "TargetService" option be set, found none`)
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
