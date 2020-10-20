package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path"
	"strings"
)

type openshiftCIAddBranchCmdFlags struct {
	branch  string
	repoDir string
}

type openshiftCIAddBranchCmd struct {
	branch  string
	repoDir string
}

func (c *openshiftCIAddBranchCmd) DoOpenShiftReleaseAddBranch(ctx context.Context) error {

	//Update CI Operator Config
	err := c.updateCIOperatorConfig(c.repoDir, c.branch)
	if err != nil {
		return err
	}

	//Update Image Mirror Mapping Config
	err = c.updateImageMirroringConfig(c.repoDir, c.branch)
	if err != nil {
		return err
	}
	return nil
}

func (c *openshiftCIAddBranchCmd) updateCIOperatorConfig(repoDir string, branch string) error {
	masterConfig := path.Join(repoDir, "ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-master.yaml")
	releaseConfig := path.Join(repoDir, fmt.Sprintf("ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-%s.yaml", branch))

	y, err := utils.LoadUnstructYaml(masterConfig)
	if err != nil {
		return err
	}

	err = y.Set("promotion.name", branch)
	if err != nil {
		return err
	}

	err = y.Set("zz_generated_metadata.branch", branch)
	if err != nil {
		return err
	}

	err = y.Write(releaseConfig)
	if err != nil {
		return err
	}

	makeExecutable, err := exec.LookPath("make")
	if err != nil {
		return err
	}

	makeJobCmd := &exec.Cmd{
		Dir:    repoDir,
		Path:   makeExecutable,
		Args:   []string{makeExecutable, "jobs"},
		Stdout: os.Stdout,
		Stderr: os.Stdout,
	}

	err = makeJobCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (c *openshiftCIAddBranchCmd) updateImageMirroringConfig(repoDir string, branch string) error {
	mappingFile := path.Join(repoDir, fmt.Sprintf("core-services/image-mirroring/integr8ly/mapping_integr8ly_operator_%s", strings.ReplaceAll(branch, ".", "_")))

	internalReg := "registry.svc.ci.openshift.org/integr8ly"
	publicReg := "quay.io/integreatly"

	type imageTemplate struct {
		internalRegTemplate string
		externalRegTemplate string
	}

	imageTemplates := []imageTemplate{
		{
			internalRegTemplate: "%s/%s:integreatly-operator",
			externalRegTemplate: "%s/integreatly-operator:%s",
		},
		{
			internalRegTemplate: "%s/%s:integreatly-operator-test-harness",
			externalRegTemplate: "%s/integreatly-operator-test-harness:%s",
		},
	}

	file, err := os.Create(mappingFile)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)

	for _, t := range imageTemplates {
		internalImage := fmt.Sprintf(t.internalRegTemplate, internalReg, branch)
		publicImage := fmt.Sprintf(t.externalRegTemplate, publicReg, branch)
		mapping := fmt.Sprintf("%s %s\n", internalImage, publicImage)
		w.WriteString(mapping)
	}

	return w.Flush()
}

func init() {

	f := &openshiftCIAddBranchCmdFlags{}

	cmd := &cobra.Command{
		Use:   "add-branch",
		Short: "Add CI Configuration for a given branch name",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := newOpenshiftCIAddBranchCmd(f)
			if err != nil {
				handleError(err)
			}
			err = c.DoOpenShiftReleaseAddBranch(cmd.Context())
			if err != nil {
				handleError(err)
			}
		},
	}

	openshifCICmd.AddCommand(cmd)
	cmd.Flags().StringVar(&f.branch, "branch", "", "Branch name")
	cmd.MarkFlagRequired("branch")
	cmd.Flags().StringVar(&f.repoDir, "repo-dir", "", "Repo Dir")
	cmd.MarkFlagRequired("repo-dir")
}

func newOpenshiftCIAddBranchCmd(f *openshiftCIAddBranchCmdFlags) (*openshiftCIAddBranchCmd, error) {
	return &openshiftCIAddBranchCmd{
		branch:  f.branch,
		repoDir: f.repoDir,
	}, nil
}
