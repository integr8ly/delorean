package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/integr8ly/delorean/pkg/polarion"
	"github.com/integr8ly/delorean/pkg/utils"
)

type polarionSessionMock struct {
	getPlanByID    func(projectID, id string) (*polarion.Plan, error)
	createPlan     func(projectID, name, id, parentID, templateID string) error
	getTestRunByID func(projectID, id string) (*polarion.TestRun, error)
	createTestRun  func(projectID, id, templateID string) (string, error)
	updateTestRun  func(uri string, title string, isTemplate bool, plannedIn string) error
}

func (s *polarionSessionMock) GetPlanByID(projectID, id string) (*polarion.Plan, error) {
	return s.getPlanByID(projectID, id)
}
func (s *polarionSessionMock) CreatePlan(projectID, name, id, parentID, templateID string) error {
	return s.createPlan(projectID, name, id, parentID, templateID)
}
func (s *polarionSessionMock) GetTestRunByID(projectID, id string) (*polarion.TestRun, error) {
	return s.getTestRunByID(projectID, id)
}
func (s *polarionSessionMock) CreateTestRun(projectID, id, templateID string) (string, error) {
	return s.createTestRun(projectID, id, templateID)
}
func (s *polarionSessionMock) UpdateTestRun(uri string, title string, isTemplate bool, plannedIn string) error {
	return s.updateTestRun(uri, title, isTemplate, plannedIn)
}

func TestPolarionRelease(t *testing.T) {

	cases := []struct {
		version     string
		session     func(t *testing.T) (polarion.PolarionSessionService, func())
		expectError bool
	}{
		{
			version: "2.1.0-rc1",
			session: func(t *testing.T) (polarion.PolarionSessionService, func()) {
				// Test that all mocked functions are called the right amount of times
				calls := 0
				return &polarionSessionMock{
						getPlanByID:    func(_, _ string) (*polarion.Plan, error) { calls++; return &polarion.Plan{}, nil },
						createPlan:     func(_, _, _, _, _ string) error { calls++; return nil },
						getTestRunByID: func(_, _ string) (*polarion.TestRun, error) { calls++; return &polarion.TestRun{}, nil },
						createTestRun:  func(_, _, _ string) (string, error) { calls++; return "uri", nil },
						updateTestRun:  func(_, _ string, _ bool, _ string) error { calls++; return nil },
					}, func() {
						if expected := 7; expected != calls {
							t.Fatalf("the session service should be called %d times but was called %d times", expected, calls)
						}
					}
			},
			expectError: false,
		},
		{
			version: "2.1.0-rc1",
			session: func(t *testing.T) (polarion.PolarionSessionService, func()) {
				// Test an error in the polarion api
				return &polarionSessionMock{
					getPlanByID: func(_, _ string) (*polarion.Plan, error) { return &polarion.Plan{}, nil },
					createPlan:  func(_, _, _, _, _ string) error { return errors.New("some error") },
				}, func() {}
			},
			expectError: true,
		},
		{
			version: "2.1.0",
			session: func(t *testing.T) (polarion.PolarionSessionService, func()) {
				// The paln should not be created if the version is not a pre-release
				calls := 0
				return &polarionSessionMock{
						getPlanByID:    func(_, _ string) (*polarion.Plan, error) { calls++; return &polarion.Plan{}, nil },
						createPlan:     func(_, _, _, _, _ string) error { calls++; return nil },
						getTestRunByID: func(_, _ string) (*polarion.TestRun, error) { calls++; return &polarion.TestRun{}, nil },
						createTestRun:  func(_, _, _ string) (string, error) { calls++; return "uri", nil },
						updateTestRun:  func(_, _ string, _ bool, _ string) error { calls++; return nil },
					}, func() {
						if expected := 0; expected != calls {
							t.Fatalf("the session service should be called %d times but was called %d times", expected, calls)
						}
					}
			},
			expectError: false,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("test polarion release for version %s", c.version), func(t *testing.T) {

			// Prepare the version
			version, err := utils.NewRHMIVersion(c.version)
			if err != nil {
				t.Fatal(err)
			}

			session, verify := c.session(t)

			// Create the polarionReleaseCmd object
			cmd := &polarionReleaseCmd{
				version:  version,
				polarion: session,
			}

			// Run the polarionReleaseCmd
			err = cmd.run()
			if !c.expectError && err != nil {
				t.Fatalf("polarionReleaseCmd failed with error: %s", err)
			} else if c.expectError && err == nil {
				t.Fatalf("polarionReleaseCmd should fails but it succed")
			}

			// Run the verify method for the mocked api
			verify()
		})
	}
}
