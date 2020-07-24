package cmd

import (
	"context"
	"github.com/integr8ly/delorean/pkg/utils"
	"testing"
)

func TestCheckOLMGraphCmd(t *testing.T) {
	cases := []struct {
		description string
		csvs        utils.CSVNames
		expectError bool
	}{
		{
			description: "success",
			csvs: utils.CSVNames{
				{
					Name:     "operator.v1.0.1",
					Replaces: "operator.v1.0.0",
				},
			},
			expectError: false,
		},
		{
			description: "success",
			csvs: utils.CSVNames{
				{
					Name:     "operator.v1.0.1",
					Replaces: "",
				},
				{
					Name:     "operator.v1.0.2",
					Replaces: "operator.v1.0.1",
				},
				{
					Name:     "operator.v1.0.3",
					Replaces: "operator.v1.0.2",
				},
			},
			expectError: false,
		},
		{
			description: "success",
			csvs: utils.CSVNames{
				{
					Name:     "operator.v1.0.1",
					Replaces: "",
				},
				{
					Name:     "operator.v1.0.2",
					Replaces: "operator.v1.0.1",
				},
				{
					Name:     "operator.v1.0.3",
					Replaces: "operator.v1.0.1",
				},
			},
			expectError: false,
		},
		{
			description: "failure",
			csvs: utils.CSVNames{
				{
					Name:     "operator.v1.0.1",
					Replaces: "operator.v1.0.0",
				},
				{
					Name:     "operator.v1.0.2",
					Replaces: "operator.v1.0.0",
				},
				{
					Name:     "operator.v1.0.4",
					Replaces: "operator.v1.0.3",
				},
			},
			expectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			cmd := &checkOLMGraphCmd{
				csvs: map[string]utils.CSVNames{
					"product": c.csvs,
				},
				directory: "/tmp/manifests",
			}
			err := cmd.run(context.TODO())
			if err != nil && !c.expectError {
				t.Fatalf("unexpected error %v", err)
			}
			if err == nil && c.expectError {
				t.Fatal("error expected but got nil")
			}
		})
	}
}
