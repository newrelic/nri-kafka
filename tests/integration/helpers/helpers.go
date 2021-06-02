package helpers

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/log"
)

const cmdMinLength = 3

// ExecInContainer executes the given command inside the specified container. It returns three values:
// 1st - Standard Output
// 2nd - Standard Error
// 3rd - Runtime error, if any
func ExecInContainer(container string, command []string) (string, string, error) {
	cmdLine := make([]string, 0, cmdMinLength+len(command))
	cmdLine = append(cmdLine, "exec", "-i")
	cmdLine = append(cmdLine, container)
	cmdLine = append(cmdLine, command...)

	log.Debug("executing: docker %s", strings.Join(cmdLine, " "))

	cmd := exec.Command("docker", cmdLine...)

	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	err := cmd.Run()
	stdout := outbuf.String()
	stderr := errbuf.String()

	return stdout, stderr, err
}
