package polarion

import "fmt"

func (s *Session) GetPlanByID(projectID, id string) (*Plan, error) {

	request := NewGetPlanByIDRequest(projectID, id)

	response := &GetPlanByIDResponseBody{}

	err := s.request(planningService, request, response)
	if err != nil {
		return nil, err
	}

	if response.Fault.Faultcode != "" {
		return nil, fmt.Errorf("getPlanById request failed with error: %s", response.Fault.Faultstring)
	}

	return &response.Plan, nil

}

func (s *Session) CreatePlan(projectID, name, id, parentID, templateID string) error {

	request := NewCreatePlanRequest(projectID, name, id, parentID, templateID)

	response := &CreatePlanResponseBody{}

	err := s.request(planningService, request, response)
	if err != nil {
		return err
	}

	if response.Fault.Faultcode != "" {
		return fmt.Errorf("createPlan request failed with error: %s", response.Fault.Faultstring)
	}

	return nil
}
