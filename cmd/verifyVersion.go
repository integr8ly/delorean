package cmd

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

type verifyVersionFlags struct {
	incomingManifests string
	currentManifests  string
}

func init() {

	flags := &verifyVersionFlags{}

	cmd := &cobra.Command{
		Use:   "verify-csv-version",
		Short: "Compare the incoming CSV version from the incoming manifests dir with the current CSV version",
		Run: func(cmd *cobra.Command, args []string) {
			err := doVerifyVersion(flags)
			if err != nil {
				handleError(err)
			}
		},
	}

	ewsCmd.AddCommand(cmd)

	cmd.Flags().StringVar(&flags.incomingManifests, "incoming-manifests", "", "the manifests directory with the incoming CSV that must be bigger or equal then the current CSV")
	cmd.MarkFlagRequired("incoming-manifests")

	cmd.Flags().StringVar(&flags.currentManifests, "current-manifests", "", "the manifests directory with the current CSV")
	cmd.MarkFlagRequired("current-manifests")
}

func doVerifyVersion(flags *verifyVersionFlags) error {

	incoming, err := getCurrentVersion(flags.incomingManifests)
	if err != nil {
		return err
	}

	current, err := getCurrentVersion(flags.currentManifests)
	if err != nil {
		return err
	}

	fmt.Printf("comparing incoming CSV version %s with current version %s\n", incoming, current)
	o := incoming.Compare(current)
	switch o {
	case -1:
		return fmt.Errorf("the incoming operator version '%s' is smaller than current version '%s'", incoming, current)
	case 0, 1:
		return nil
	}
	return fmt.Errorf("unexpected compare result: %d", o)
}

func getCurrentVersion(manifests string) (semver.Version, error) {
	csv, _, err := utils.GetCurrentCSV(manifests)
	if err != nil {
		return semver.Version{}, err
	}

	return csv.GetVersion()
}
