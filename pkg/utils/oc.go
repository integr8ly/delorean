package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type OCInterface interface {
	Run(arg ...string) error
	RunWithOutputFile(outputFile string, arg ...string) error
}

type OC struct {
	Kubeconfig string
	executable string
}

func NewOC(kubeconfig string) *OC {
	return &OC{Kubeconfig: kubeconfig, executable: "oc"}
}

// Run oc with the given args. Outputs are written to stdout
func (r *OC) Run(arg ...string) error {
	return r.run(os.Stdout, arg...)
}

// Run oc with the given args, outputs are written to the given outputFile
func (r *OC) RunWithOutputFile(outputFile string, arg ...string) error {
	if outputFile == "" {
		return errors.New("output file is empty")
	}
	outfile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outfile.Close()
	return r.run(outfile, arg...)
}

func (r *OC) run(out io.Writer, arg ...string) error {
	oc, err := exec.LookPath(r.executable)
	if err != nil {
		return nil
	}
	envs := []string{fmt.Sprintf("KUBECONFIG=%s", r.Kubeconfig)}
	args := []string{oc}
	args = append(args, arg...)
	c := &exec.Cmd{
		Path:   oc,
		Args:   args,
		Stdout: out,
		Stderr: os.Stderr,
		Env:    envs,
	}
	return c.Run()
}
