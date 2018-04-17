package monitor

import (
	"bytes"
	l4g "github.com/alecthomas/log4go"
	"io/ioutil"
	"os/exec"
	"syscall"
)

type ScriptResult struct {
	Filename  string
	ExitCode  int64
	SystemOut string
}

type MonitorScript struct {
	ScriptsDirectory string
}

func NewMonitorScript(scriptDir string) *MonitorScript {
	ms := new(MonitorScript)
	ms.ScriptsDirectory = scriptDir
	return ms
}

func (ms *MonitorScript) GetScriptResult() ([]*ScriptResult, error) {
	dir := ms.ScriptsDirectory
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var results []*ScriptResult = []*ScriptResult{}
	for _, file := range files {
		l4g.Debug("Exec script: " + file.Name())
		cmd := exec.Command(dir + "/" + file.Name())
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		err := cmd.Start()
		if err != nil {
			l4g.Error(err)
		}

		result := new(ScriptResult)
		result.Filename = file.Name()
		if err := cmd.Wait(); err != nil {
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					l4g.Debug("Exit Status: %d", status.ExitStatus())
					result.ExitCode = int64(status.ExitStatus())
				}
			} else {
				l4g.Warn("cmd.Wait: %v", err)
				result.ExitCode = 0
			}
		} else {
			l4g.Debug("No error")
			result.ExitCode = 0
		}

		l4g.Debug("Output: " + stdout.String())
		result.SystemOut = stdout.String()
		results = append(results, result)
	}

	return results, nil
}
