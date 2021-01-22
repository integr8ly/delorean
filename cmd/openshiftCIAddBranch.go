package cmd

import (
	"bufio"
	"fmt"
	"github.com/integr8ly/delorean/pkg/types"
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
	olmType string
}

type openshiftCIAddBranchCmd struct {
	branch  string
	repoDir string
	olmType string
}

func (c *openshiftCIAddBranchCmd) DoOpenShiftReleaseAddBranch() error {

	//Update CI Operator Config
	err := c.updateCIOperatorConfig()
	if err != nil {
		return err
	}

	//Update Image Mirror Mapping Config
	err = c.updateImageMirroringConfig()
	if err != nil {
		return err
	}
	return nil
}

func (c *openshiftCIAddBranchCmd) updateCIOperatorConfig() error {
	var configFile string
	// olmType controls the prow config which is copied
	// in the case of RHOAM we don't need to run e2e tests for release branch as it will only be used for a rhoam patch
	// release and for RHMI we don't need to run rhoam-e2e tests
	// using the earliest version of each RHOAM and RHMI release (1.1 and 2.0 respectively)
	// if neither RHOAM or RHMI types are found use the master config as a default
	// all config files can be found here -> https://github.com/openshift/release/tree/master/ci-operator/config/integr8ly/integreatly-operator
	switch olmType {
	case types.OlmTypeRhmi:
		configFile = "ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-release-v2.0.yaml"
	case types.OlmTypeRhoam:
		configFile = "ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-rhoam-release-v1.1.yaml"
	default:
		configFile = "ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-master.yaml"
	}
	masterConfig := path.Join(c.repoDir, configFile)
	releaseConfig := path.Join(c.repoDir, fmt.Sprintf("ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-%s.yaml", c.branch))

	y, err := utils.LoadUnstructYaml(masterConfig)
	if err != nil {
		return err
	}

	err = y.Set("promotion.name", c.branch)
	if err != nil {
		return err
	}

	err = y.Set("zz_generated_metadata.branch", c.branch)
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
		Dir:    c.repoDir,
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

func (c *openshiftCIAddBranchCmd) updateImageMirroringConfig() error {
	mappingFile := path.Join(c.repoDir, fmt.Sprintf("core-services/image-mirroring/integr8ly/mapping_integr8ly_operator_%s", strings.ReplaceAll(c.branch, ".", "_")))

	internalReg := "registry.ci.openshift.org/integr8ly"
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
		internalImage := fmt.Sprintf(t.internalRegTemplate, internalReg, c.branch)
		publicImage := fmt.Sprintf(t.externalRegTemplate, publicReg, c.branch)
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
			err = c.DoOpenShiftReleaseAddBranch()
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
	cmd.Flags().StringVar(&f.olmType, "olmType", "", "OLM type for the release. Valid inputs are \"integreatly-operator\" or \"managed-api-service\"")
	cmd.MarkFlagRequired("olmType")
}

func newOpenshiftCIAddBranchCmd(f *openshiftCIAddBranchCmdFlags) (*openshiftCIAddBranchCmd, error) {
	return &openshiftCIAddBranchCmd{
		branch:  f.branch,
		repoDir: f.repoDir,
		olmType: f.olmType,
	}, nil
}
