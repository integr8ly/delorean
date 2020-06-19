package polarion

import "encoding/xml"

type service string

const (
	sessionService        service = "SessionWebService"
	planningService       service = "PlanningWebService"
	testManagementService service = "TestManagementWebService"
)

type FaultBody struct {
	Fault Fault `xml:"Fault"`
}

type Fault struct {
	Faultcode   string `xml:"faultcode"`
	Faultstring string `xml:"faultstring"`
}

type SessionRequest struct {
	XMLName xml.Name             `xml:"soapenv:Envelope"`
	SoapNS  string               `xml:"xmlns:soapenv,attr"`
	Header  SessionRequestHeader `xml:"soapenv:Header"`
	Body    interface{}          `xml:"soapenv:Body"`
}

type SessionRequestHeader struct {
	SesNS     string `xml:"xmlns:ses,attr"`
	SessionID string `xml:"ses:sessionID"`
}

type SessionResponse struct {
	Header struct{}    `xml:"Header"`
	Body   interface{} `xml:"Body"`
}

type LogInEnvelopResponse struct {
	XMLName xml.Name            `xml:"Envelope"`
	Header  LogInResponseHeader `xml:"Header"`
	Body    FaultBody           `xml:"Body"`
}

type LogInResponseHeader struct {
	SessionID string `xml:"sessionID"`
}

type LogInRequest struct {
	XMLName xml.Name         `xml:"soapenv:Envelope"`
	SoapNS  string           `xml:"xmlns:soapenv,attr"`
	SesNS   string           `xml:"xmlns:ses,attr"`
	Header  struct{}         `xml:"soapenv:Header"`
	Body    LogInRequestBody `xml:"soapenv:Body"`
}

type LogInRequestBody struct {
	LogIn LogInRequestBodyLogIn `xml:"ses:logIn"`
}

type LogInRequestBodyLogIn struct {
	UserName string `xml:"ses:userName"`
	Password string `xml:"ses:password"`
}

type GetPlanByIDRequestBody struct {
	PlanNS      string             `xml:"xmlns:plan,attr"`
	GetPlanByID GetPlanByIDRequest `xml:"plan:getPlanById"`
}

type GetPlanByIDRequest struct {
	ProjectID string `xml:"plan:projectId"`
	ID        string `xml:"plan:id"`
}

type GetPlanByIDResponseBody struct {
	Plan Plan `xml:"getPlanByIdResponse>getPlanByIdReturn"`
	FaultBody
}

type Plan struct {
	ID         string `xml:"id"`
	IsTemplate bool   `xml:"isTemplate"`
	Name       string `xml:"name"`
	Status     Status `xml:"status"`
	Parent     *Plan  `xml:"parent"`
}

type Status struct {
	ID string `xml:"id"`
}

type CreatePlanRequestBody struct {
	PlanNS     string            `xml:"xmlns:plan,attr"`
	CreatePlan CreatePlanRequest `xml:"plan:createPlan"`
}

type CreatePlanRequest struct {
	ProjectID  string `xml:"plan:projectId"`
	Name       string `xml:"plan:name"`
	ID         string `xml:"plan:id"`
	ParentID   string `xml:"plan:parentId,omitempty"`
	TemplateID string `xml:"plan:templateId"`
}

type CreatePlanResponseBody struct {
	FaultBody
}

type GetTestRunByIDRequestBody struct {
	TesNS          string                `xml:"xmlns:tes,attr"`
	GetTestRunByID GetTestRunByIDRequest `xml:"tes:getTestRunById"`
}

type GetTestRunByIDRequest struct {
	ProjectID string `xml:"tes:projectId"`
	ID        string `xml:"tes:id"`
}

type GetTestRunByIDResponseBody struct {
	TestRun TestRun `xml:"getTestRunByIdResponse>getTestRunByIdReturn"`
	FaultBody
}

type TestRun struct {
	ID         string `xml:"id"`
	IsTemplate bool   `xml:"isTemplate"`
	Title      string `xml:"title"`
	Status     Status `xml:"status"`
}

type CreateTestRunRequestBody struct {
	TesNS         string               `xml:"xmlns:tes,attr"`
	CreateTestRun CreateTestRunRequest `xml:"tes:createTestRun"`
}

type CreateTestRunRequest struct {
	Project  string `xml:"tes:project"`
	ID       string `xml:"tes:id"`
	Template string `xml:"tes:template,omitempty"`
}

type CreateTestRunResponseBody struct {
	URI string `xml:"createTestRunResponse>createTestRunReturn"`
	FaultBody
}

type UpdateTestRunRequestBody struct {
	TesNS         string               `xml:"xmlns:tes,attr"`
	Tes1NS        string               `xml:"xmlns:tes1,attr"`
	TracNS        string               `xml:"xmlns:trac,attr"`
	UpdateTestRun UpdateTestRunRequest `xml:"tes:updateTestRun>tes:content"`
}

type UpdateTestRunRequest struct {
	URI          string                            `xml:"uri,attr"`
	Title        string                            `xml:"tes1:title,omitempty"`
	IsTemplate   bool                              `xml:"tes1:isTemplate,omitempty"`
	CustomFields []UpdateTestRunRequestCustomField `xml:"tes1:customFields>trac:Custom,omitempty"`
}

type UpdateTestRunRequestCustomField struct {
	Key   string      `xml:"trac:key"`
	Value interface{} `xml:"trac:value"`
}

type EnumOptionId struct {
	ID string `xml:"trac:id"`
}

type UpdateTestRunResponseBody struct {
	FaultBody
}

func NewSessionRequest(sessionID string, body interface{}) *SessionRequest {
	return &SessionRequest{
		SoapNS: "http://schemas.xmlsoap.org/soap/envelope/",
		Header: SessionRequestHeader{
			SesNS:     "http://ws.polarion.com/session",
			SessionID: sessionID,
		},
		Body: body,
	}
}

func NewLogInRequest(s LogInRequestBodyLogIn) *LogInRequest {
	return &LogInRequest{
		SoapNS: "http://schemas.xmlsoap.org/soap/envelope/",
		SesNS:  "http://ws.polarion.com/SessionWebService-impl",
		Header: struct{}{},
		Body:   LogInRequestBody{LogIn: s},
	}
}

func NewGetPlanByIDRequest(projectID, id string) *GetPlanByIDRequestBody {
	return &GetPlanByIDRequestBody{
		PlanNS: "http://ws.polarion.com/PlanningWebService-impl",
		GetPlanByID: GetPlanByIDRequest{
			ProjectID: projectID,
			ID:        id,
		},
	}
}

func NewCreatePlanRequest(projectID, name, id, parentID, templateID string) *CreatePlanRequestBody {
	return &CreatePlanRequestBody{
		PlanNS: "http://ws.polarion.com/PlanningWebService-impl",
		CreatePlan: CreatePlanRequest{
			ProjectID:  projectID,
			Name:       name,
			ID:         id,
			ParentID:   parentID,
			TemplateID: templateID,
		},
	}
}

func NewGetTestRunByIDRequest(projectID, id string) *GetTestRunByIDRequestBody {
	return &GetTestRunByIDRequestBody{
		TesNS: "http://ws.polarion.com/TestManagementWebService-impl",
		GetTestRunByID: GetTestRunByIDRequest{
			ProjectID: projectID,
			ID:        id,
		},
	}
}

func NewCreateTestRunRequest(projectID, id, templateID string) *CreateTestRunRequestBody {
	return &CreateTestRunRequestBody{
		TesNS: "http://ws.polarion.com/TestManagementWebService-impl",
		CreateTestRun: CreateTestRunRequest{
			Project:  projectID,
			ID:       id,
			Template: templateID,
		},
	}
}

func NewUpdateTestRunRequest(uri string, title string, isTemplate bool, plannedIn string) *UpdateTestRunRequestBody {

	c := []UpdateTestRunRequestCustomField{}

	if plannedIn != "" {
		c = append(c, UpdateTestRunRequestCustomField{
			Key:   "plannedin",
			Value: EnumOptionId{ID: plannedIn},
		})
	}

	return &UpdateTestRunRequestBody{
		TesNS:  "http://ws.polarion.com/TestManagementWebService-impl",
		Tes1NS: "http://ws.polarion.com/TestManagementWebService-types",
		TracNS: "http://ws.polarion.com/TrackerWebService-types",
		UpdateTestRun: UpdateTestRunRequest{
			URI:          uri,
			Title:        title,
			IsTemplate:   isTemplate,
			CustomFields: c,
		},
	}
}
