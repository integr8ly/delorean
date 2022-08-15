package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"io/ioutil"
	"path"
)

const (
	baseKeycloakV18 = "keycloak-operator.v18.0.0"
	baseKeycloakV9  = "keycloak-operator.v9.0.3"
)

type checkOLMGraphFlags struct {
	directory string
}

type checkOLMGraphCmd struct {
	directory string
	csvs      map[string]utils.CSVNames
}

func init() {
	f := &checkOLMGraphFlags{}
	cmd := &cobra.Command{
		Use:   "check-olm-graph",
		Short: "Check if the OLM graph chain is broken for a given OLM manifest directory",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := newCheckOLMGraphCmd(f)
			if err != nil {
				handleError(err)
			}
			if err := c.run(cmd.Context()); err != nil {
				handleError(err)
			}
		},
	}

	ewsCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&f.directory, "directory", "d", "", "Path to the OLM manifest directory")
	cmd.MarkFlagRequired("directory")
}

func newCheckOLMGraphCmd(f *checkOLMGraphFlags) (*checkOLMGraphCmd, error) {
	csvs := make(map[string]utils.CSVNames)
	dirs, err := ioutil.ReadDir(f.directory)
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		if d.IsDir() {
			p := path.Join(f.directory, d.Name())
			c, err := utils.GetSortedCSVNames(p)
			if err != nil {
				return nil, err
			}
			csvs[d.Name()] = c
		}
	}

	return &checkOLMGraphCmd{
		directory: f.directory,
		csvs:      csvs,
	}, err
}

func (c *checkOLMGraphCmd) run(ctx context.Context) error {
	var hasError bool
	for dir, csvs := range c.csvs {
		err := checkGraphInDir(dir, csvs)
		if err != nil {
			hasError = true
		}
	}
	if hasError {
		return errors.New(fmt.Sprintf("OLM graph check failed in %s", c.directory))
	}
	return nil
}

func checkGraphInDir(dirname string, csvs utils.CSVNames) error {
	if csvs.Len() <= 1 {
		fmt.Println(fmt.Sprintf("[%s] no graph to check", dirname))
		return nil
	}
	for i := csvs.Len() - 1; i > 0; i-- {
		csv := csvs[i]
		if !csvs.Contains(csv.Replaces) && (csv.Name != baseKeycloakV18) && (csv.Name != baseKeycloakV9) {
			fmt.Println(fmt.Sprintf("[%s] OLM graph is broken. CSV %s replaces %s, which doesn't exist", dirname, csv.Name, csv.Replaces))
			return errors.New(fmt.Sprintf("[%s] invalid replaces field %s in CSV %s", dirname, csv.Replaces, csv.Name))
		}
	}
	fmt.Println(fmt.Sprintf("[%s] OLM graph is complete", dirname))
	return nil
}
