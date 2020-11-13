package cmd

import (
	"fmt"

	"github.com/integr8ly/delorean/pkg/polarion"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

var polarionProjectIDs = map[string]string{
	"rhmi":  "RedHatManagedIntegration",
	"rhoam": "OpenShiftAPIManagement",
}

const (
	polarionServicesURL        = "https://polarion.engineering.redhat.com/polarion/ws/services"
	polarionServicesStagingURL = "https://polarion.stage.engineering.redhat.com/polarion/ws/services"
)

type polarionReleaseFlags struct {
	productName string
	version     string
	stage       bool
	debug       bool
}

type polarionReleaseCmd struct {
	projectID string
	version   *utils.RHMIVersion
	polarion  polarion.PolarionSessionService
}

func newPolarionReleaseCmd(f *polarionReleaseFlags) (*polarionReleaseCmd, error) {

	projectID := polarionProjectIDs[f.productName]
	if projectID == "" {
		return nil, fmt.Errorf("%s is not a valid productName. See the usage for supported product names", f.productName)
	}

	version, err := utils.NewRHMIVersion(f.version)
	if err != nil {
		return nil, err
	}

	polarionUsername, err := requireValue(PolarionUsernameKey)
	if err != nil {
		return nil, err
	}

	polarionPassword, err := requireValue(PolarionPasswordKey)
	if err != nil {
		return nil, err
	}

	var url string
	if f.stage {
		url = polarionServicesStagingURL
	} else {
		url = polarionServicesURL
	}

	polarion, err := polarion.NewSession(polarionUsername, polarionPassword, url, f.debug)
	if err != nil {
		return nil, err
	}

	return &polarionReleaseCmd{
		projectID: projectID,
		version:   version,
		polarion:  polarion,
	}, nil
}

func init() {
	f := &polarionReleaseFlags{}

	cmd := &cobra.Command{
		Use:   "polarion-release",
		Short: "Prepare the release version in Polarion",
		Run: func(cmd *cobra.Command, args []string) {

			c, err := newPolarionReleaseCmd(f)
			if err != nil {
				handleError(err)
			}

			err = c.run()
			if err != nil {
				handleError(err)
			}
		},
	}

	reportCmd.AddCommand(cmd)

	cmd.Flags().StringVar(&f.productName, "product-name", "", "Name of the product. Valid inputs are: \"rhmi\", \"rhoam\"")
	cmd.MarkFlagRequired("product-name")
	cmd.Flags().StringVar(&f.version, "version", "", "The RHMI/RHOAM version to create in Polarion (ex \"2.0.0\", \"2.0.0-er4\")")
	cmd.MarkFlagRequired("version")

	cmd.Flags().BoolVar(&f.stage, "stage", false, "Create the release in the Polarion staging environment")
	cmd.Flags().BoolVar(&f.debug, "debug", false, "Print the Polarion API request and response")

}

func (c *polarionReleaseCmd) run() error {

	if !c.version.IsPreRelease() {
		fmt.Println("skip non pre-release", c.version.String())
		return nil
	}

	err := c.creteRelease()
	if err != nil {
		return fmt.Errorf("failed to create the release for %s with error: %s", c.version, err)
	}

	err = c.creteMilestone()
	if err != nil {
		return fmt.Errorf("failed to create the milestone for %s with error: %s", c.version, err)
	}

	err = c.createTestRunTemplate()
	if err != nil {
		return fmt.Errorf("failed to create the test run template for %s with error: %s", c.version, err)
	}

	return nil
}

func (c *polarionReleaseCmd) creteRelease() error {

	id := c.version.PolarionReleaseId()

	plan, err := c.polarion.GetPlanByID(c.projectID, id)
	if err != nil {
		return err
	}

	if plan.ID != "" {
		fmt.Printf("the release '%s' already exists\n", plan.ID)
		return nil
	}

	err = c.polarion.CreatePlan(
		c.projectID,
		c.version.MajorMinorPatch(),
		id,
		"",
		"release",
	)
	if err != nil {
		return err
	}

	fmt.Printf("the release '%s' has been created\n", id)
	return nil
}

func (c *polarionReleaseCmd) creteMilestone() error {

	id := c.version.PolarionMilestoneId()

	plan, err := c.polarion.GetPlanByID(c.projectID, id)
	if err != nil {
		return err
	}

	if plan.ID != "" {
		fmt.Printf("the milestone '%s' already exists\n", plan.ID)
		return nil
	}

	err = c.polarion.CreatePlan(
		c.projectID,
		c.version.String(),
		id,
		c.version.PolarionReleaseId(),
		"Beta",
	)
	if err != nil {
		return err
	}

	fmt.Printf("the milestone '%s' has been created\n", id)
	return nil
}

func (c *polarionReleaseCmd) createTestRunTemplate() error {

	id := c.version.PolarionMilestoneId()

	template, err := c.polarion.GetTestRunByID(c.projectID, id)
	if err != nil {
		return err
	}

	if template.ID != "" {
		fmt.Printf("the test run template '%s' already exists\n", template.ID)
		return nil
	}

	uri, err := c.polarion.CreateTestRun(
		c.projectID,
		id,
		"XUnit",
	)
	if err != nil {
		return err
	}

	title := fmt.Sprintf("%s Template", c.version.String())
	c.polarion.UpdateTestRun(uri, title, true, id)

	fmt.Printf("the test run template '%s' has been created\n", id)
	return nil
}
