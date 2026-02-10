package test

import (
	"context"
	"os/exec"

	"github.com/plgd-dev/kit/v2/codec/json"
)

func RestartDockerContainer(dockerName string) error {
	cmd := exec.CommandContext(context.Background(), "docker", "restart", dockerName)
	return cmd.Run()
}

func SendSignalToDocker(dockerName, signal string) error {
	cmd := exec.CommandContext(context.Background(), "docker", "kill", "-s", signal, dockerName)
	return cmd.Run()
}

func SendSignalToProcess(pid, signal string) error {
	cmd := exec.CommandContext(context.Background(), "kill", "-s", signal, pid)
	return cmd.Run()
}

func ToJSON(v interface{}) ([]byte, error) {
	data, err := json.Encode(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}
