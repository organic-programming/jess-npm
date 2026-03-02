package npmrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
)

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Elapsed  float64
}

func RunNpm(args []string, workdir string, env []string, timeoutS int) Result {
	return runCommand("npm", args, workdir, env, timeoutS)
}

func RunNpmJSON(args []string, workdir string, env []string, timeoutS int) (Result, []byte) {
	jsonArgs := ensureJSON(args)
	result := RunNpm(jsonArgs, workdir, env, timeoutS)
	return result, []byte(result.Stdout)
}

func RunNode(args []string, workdir string, env []string, timeoutS int) Result {
	return runCommand("node", args, workdir, env, timeoutS)
}

func RunNodeEval(code string, workdir string, env []string, timeoutS int) Result {
	return RunNode([]string{"-e", code}, workdir, env, timeoutS)
}

func runCommand(binary string, args []string, workdir string, env []string, timeoutS int) Result {
	ctx := context.Background()
	cancel := func() {}
	if timeoutS > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutS)*time.Second)
	}
	defer cancel()

	if workdir == "" {
		workdir = "."
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), env...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	res := Result{
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
		Elapsed: time.Since(start).Seconds(),
	}

	if err == nil {
		return res
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		res.ExitCode = 124
		if strings.TrimSpace(res.Stderr) == "" {
			res.Stderr = fmt.Sprintf("%s timed out after %ds", binary, timeoutS)
		}
		return res
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.ExitCode = exitErr.ExitCode()
		return res
	}

	res.ExitCode = 1
	if strings.TrimSpace(res.Stderr) == "" {
		res.Stderr = err.Error()
	}
	return res
}

func ensureJSON(args []string) []string {
	if slices.Contains(args, "--json") {
		return append([]string{}, args...)
	}
	jsonArgs := make([]string, 0, len(args)+1)
	jsonArgs = append(jsonArgs, args...)
	jsonArgs = append(jsonArgs, "--json")
	return jsonArgs
}
