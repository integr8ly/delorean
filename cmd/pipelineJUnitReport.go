package cmd

import (
	"context"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"os"
)

type pipelineJUnitReportFlags struct {
	inputFile  string
	outputFile string
	filter     string
}

type pipelineJUnitReportCmd struct {
	input  string
	output string
	filter string
}

func init() {
	f := &pipelineJUnitReportFlags{}
	cmd := &cobra.Command{
		Use:   "junit-report",
		Short: "Generate a junit report(s) from the pipeline status JSON file",
		Run: func(cmd *cobra.Command, args []string) {
			c := &pipelineJUnitReportCmd{
				input:  f.inputFile,
				output: f.outputFile,
				filter: f.filter,
			}

			if err := c.run(cmd.Context()); err != nil {
				handleError(err)
			}
		},
	}

	pipelineCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&f.inputFile, "input", "i", "", "Path to the input pipeline-status.json file")
	cmd.MarkFlagRequired("input")
	cmd.Flags().StringVarP(&f.outputFile, "output", "o", "", "Path to the output junit file")
	cmd.MarkFlagRequired("output")
	cmd.Flags().StringVarP(&f.filter, "filter", "", "", "Create a report only from the stages with given filter")
}

func (p *pipelineJUnitReportCmd) run(ctx context.Context) error {
	pr := &utils.PipelineRun{}
	if err := utils.PopulateObjectFromJSON(p.input, pr); err != nil {
		return err
	}
	suites, err := pr.ToJUnitSuites(p.filter)
	if err != nil {
		return err
	}
	o, err := os.Create(p.output)
	if err != nil {
		return err
	}
	if err := suites.WriteXML(o); err != nil {
		return err
	}
	return nil
}
