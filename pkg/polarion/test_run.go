package polarion

import "fmt"

func (s *Session) GetTestRunByID(projectID, id string) (*TestRun, error) {

	request := NewGetTestRunByIDRequest(projectID, id)

	response := &GetTestRunByIDResponseBody{}

	err := s.request(testManagementService, request, response)
	if err != nil {
		return nil, err
	}

	if response.Fault.Faultcode != "" {
		return nil, fmt.Errorf("getTestRunById request failed with error: %s", response.Fault.Faultstring)
	}

	return &response.TestRun, nil
}

func (s *Session) CreateTestRun(projectID, id, templateID string) (string, error) {

	request := NewCreateTestRunRequest(projectID, id, templateID)

	response := &CreateTestRunResponseBody{}

	err := s.request(testManagementService, request, response)
	if err != nil {
		return "", err
	}

	if response.Fault.Faultcode != "" {
		return "", fmt.Errorf("createTestRun request failed with error: %s", response.Fault.Faultstring)
	}

	return response.URI, nil
}

func (s *Session) UpdateTestRun(uri string, title string, isTemplate bool, plannedIn string) error {

	request := NewUpdateTestRunRequest(uri, title, isTemplate, plannedIn)

	response := &UpdateTestRunResponseBody{}

	err := s.request(testManagementService, request, response)
	if err != nil {
		return err
	}

	if response.Fault.Faultcode != "" {
		return fmt.Errorf("updateTestRun request failed with error: %s", response.Fault.Faultstring)
	}

	return nil
}
