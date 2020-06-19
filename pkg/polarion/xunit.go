package polarion

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"regexp"

	"github.com/jstemmer/go-junit-report/formatter"
)

const (
	XunitEndpoint    = "/xunit"
	JobQueueEndpoint = "/xunit-queue"

	idRegex = "^(?:.+/)*?([A-Z][0-9]{2})_.*$"
)

type PolarionXUnit struct {
	XMLName    xml.Name                  `xml:"testsuites"`
	Properties []formatter.JUnitProperty `xml:"properties>property,omitempty"`
	Suites     []PolarionXUnitTestSuite
}

type PolarionXUnitTestSuite struct {
	formatter.JUnitTestSuite

	TestCases []PolarionXUnitTestCase `xml:"testcase"`
}

type PolarionXUnitTestCase struct {
	formatter.JUnitTestCase

	Properties []formatter.JUnitProperty `xml:"properties>property,omitempty"`
}

type XUnitImportResponse struct {
	Files struct {
		File struct {
			JobIDs []int `json:"job-ids"`
		} `json:"file.xml"`
	} `json:"files"`
}

type XUnitJobStatus string

type XUnitJobStatusRespons struct {
	Jobs []struct {
		Status XUnitJobStatus `json:"status"`
	} `json:"jobs"`
}

const (
	ReadyStatus   XUnitJobStatus = "READY"
	RunningStatus XUnitJobStatus = "RUNNING"
	SuccessStatus XUnitJobStatus = "SUCCESS"
)

type XUnitImporterService interface {
	Import(xunit *PolarionXUnit) (int, error)
	GetJobStatus(id int) (XUnitJobStatus, error)
}

type XUnitImporter struct {
	url      string
	username string
	password string
}

func NewXUnitImporter(url, username, password string) *XUnitImporter {
	return &XUnitImporter{
		url:      url,
		username: username,
		password: password,
	}
}

func (x *XUnitImporter) request(request *http.Request, response interface{}) error {

	// set auth
	request.SetBasicAuth(x.username, x.password)

	// perform request
	r, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("upload reques failed with status %s", r.Status)
	}

	// parse response
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, response)
	if err != nil {
		return err
	}

	return nil
}

func (x *XUnitImporter) Import(xunit *PolarionXUnit) (int, error) {

	xml, err := xml.Marshal(xunit)
	if err != nil {
		return 0, err
	}

	// craete the request body
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, err := w.CreateFormFile("file", "file.xml")
	if err != nil {
		return 0, err
	}
	_, err = f.Write(xml)
	if err != nil {
		return 0, err
	}
	err = w.Close()
	if err != nil {
		return 0, err
	}

	request, err := http.NewRequest("POST", x.url+XunitEndpoint, &b)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", w.FormDataContentType())

	response := &XUnitImportResponse{}

	err = x.request(request, response)
	if err != nil {
		return 0, err
	}

	if len(response.Files.File.JobIDs) < 1 {
		return 0, fmt.Errorf("polarion xuint importer didn't return the job id")
	}

	return response.Files.File.JobIDs[0], nil
}

func (x *XUnitImporter) GetJobStatus(id int) (XUnitJobStatus, error) {

	url := fmt.Sprintf("%s%s?jobIds=%d", x.url, JobQueueEndpoint, id)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")

	response := &XUnitJobStatusRespons{}

	err = x.request(request, response)
	if err != nil {
		return "", err
	}

	if len(response.Jobs) < 1 {
		return "", fmt.Errorf("job with id %d not found", id)
	}

	return response.Jobs[0].Status, nil
}

func JUnitToPolarionXUnit(junit *formatter.JUnitTestSuites, projectID, title, templateID string) (*PolarionXUnit, error) {

	idr := regexp.MustCompile(idRegex)

	tests := []PolarionXUnitTestCase{}

	for _, t := range junit.Suites[0].TestCases {

		matches := idr.FindAllStringSubmatch(t.Name, 1)
		if matches == nil || len(matches) < 1 || len(matches[0]) < 1 {
			fmt.Println("skip:", t.Name)
		} else {

			test := PolarionXUnitTestCase{
				JUnitTestCase: t,
				Properties: []formatter.JUnitProperty{
					{Name: "polarion-testcase-id", Value: matches[0][1]},
				},
			}

			tests = append(tests, test)
		}

	}

	return &PolarionXUnit{
		Properties: []formatter.JUnitProperty{
			{Name: "polarion-project-id", Value: projectID},
			{Name: "polarion-testrun-title", Value: title},
			{Name: "polarion-testrun-template-id", Value: templateID},
			{Name: "polarion-lookup-method", Value: "custom"},
		},
		Suites: []PolarionXUnitTestSuite{{
			JUnitTestSuite: junit.Suites[0],
			TestCases:      tests,
		}},
	}, nil
}
