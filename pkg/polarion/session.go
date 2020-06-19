package polarion

type PolarionSessionService interface {
	GetPlanByID(projectID, id string) (*Plan, error)
	CreatePlan(projectID, name, id, parentID, templateID string) error
	GetTestRunByID(projectID, id string) (*TestRun, error)
	CreateTestRun(projectID, id, templateID string) (string, error)
	UpdateTestRun(uri string, title string, isTemplate bool, plannedIn string) error
}

type Session struct {
	client  *Client
	session string
}

func NewSession(username, password, url string, debug bool) (*Session, error) {

	client := NewClient(url, debug)

	response, err := client.LogIn(username, password)
	if err != nil {
		return nil, err
	}

	return &Session{
		client:  client,
		session: response.Header.SessionID,
	}, nil
}

func (s *Session) request(service service, request, response interface{}) error {

	req := NewSessionRequest(s.session, request)

	res := &SessionResponse{Body: response}

	return s.client.request(service, req, res)
}
