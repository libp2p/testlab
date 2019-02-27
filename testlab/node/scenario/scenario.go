package scenario

import (
	"fmt"

	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/testlab/utils"
)

type ScenarioNode struct{}

func (s *ScenarioNode) Task(options map[string]string) (*napi.Task, error) {
	var ok bool
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

	var serviceName string
	if serviceName, ok = options["TargetService"]; !ok {
		return nil, fmt.Errorf(`scenarios require a "TargetService" option be set, found none`)
	}

	task.Env = map[string]string{
		"SERVICE_NAME": serviceName,
	}

	return task, nil
}
