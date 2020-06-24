package polarion

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/jstemmer/go-junit-report/formatter"
)

func TestJUnitToPolarionXUnit(t *testing.T) {

	bytes, err := ioutil.ReadFile("testdata/junit.xml")
	if err != nil {
		t.Fatal(err)
	}

	junit := &formatter.JUnitTestSuites{}
	err = xml.Unmarshal(bytes, junit)
	if err != nil {
		t.Fatal(err)
	}

	xunit, err := JUnitToPolarionXUnit(junit, "RedHatManagedIntegration", "Some Tests", "XUnit")
	if err != nil {
		t.Fatalf("failed to convert junit to xunit with error: %s", err)
	}

	got, err := xml.Marshal(xunit)
	if err != nil {
		t.Fatal(err)
	}

	expected := `<testsuites><properties><property name="polarion-project-id" value="RedHatManagedIntegration"></property><property name="polarion-testrun-title" value="Some Tests"></property><property name="polarion-testrun-template-id" value="XUnit"></property><property name="polarion-lookup-method" value="custom"></property></properties><testsuite tests="7" failures="3" time="4354.820" name=""><properties></properties><testcase classname="" name="TestIntegreatly/Integreatly_Happy_Path_Tests/A01_-_Verify_that_all_stages_in_the_integreatly-operator_CR_report_completed" time="11.140"><properties><property name="polarion-testcase-id" value="A01"></property></properties></testcase><testcase classname="" name="TestIntegreatly/Integreatly_Happy_Path_Tests/A22_-_Verify_RHMI_Config_Updates_CRO_Strategy_Override_Config_Map" time="41.600"><failure message="Failed" type="">Some error</failure><properties><property name="polarion-testcase-id" value="A22"></property></properties></testcase><testcase classname="" name="TestIntegreatly/Integreatly_Happy_Path_Tests/A21_-_Verify_AWS_maintenance_and_backup_windows" time="44.990"><skipped message="Skip"></skipped><properties><property name="polarion-testcase-id" value="A21"></property></properties></testcase></testsuite></testsuites>`
	if string(got) != expected {
		t.Fatalf("got xunit result is not equal to the expected result\ngot: %s\nexpected: %s", string(got), expected)
	}
}

func TestXUnitImporterImport(t *testing.T) {
	endpoint, mux, teardown := setup()
	defer teardown()

	mux.HandleFunc(XunitEndpoint, func(w http.ResponseWriter, r *http.Request) {

		// Validate BasicAuth
		username, password, _ := r.BasicAuth()
		if expected := "test"; username != expected {
			t.Fatalf("username '%s' is not equal to the expected username '%s'", username, expected)
		}
		if expected := "passw"; password != expected {
			t.Fatalf("password '%s' is not equal to the expected password '%s'", password, expected)
		}

		err := r.ParseMultipartForm(0)
		if err != nil {
			t.Fatal(err)
		}

		f, _, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}

		got, err := ioutil.ReadAll(f)
		if err != nil {
			t.Fatal(err)
		}

		expected := `<testsuites><properties><property name="polarion-project-id" value="TestProject"></property><property name="polarion-testrun-title" value="Test Title"></property></properties><testsuite tests="2" failures="1" time="" name=""><properties></properties><testcase classname="" name="Success test case" time=""><properties><property name="polarion-testcase-id" value="Z01"></property></properties></testcase><testcase classname="" name="Failed test case" time=""><failure message="Failed" type="">Some error</failure><properties><property name="polarion-testcase-id" value="Z02"></property></properties></testcase></testsuite></testsuites>`
		if expected != string(got) {
			t.Fatalf("got xunit request is not equal to expected request\ngot: %s\nexpected: %s", string(got), expected)
		}

		fmt.Fprint(w, `{ "files": { "file.xml": { "job-ids": [99] } } }`)
	})

	x := NewXUnitImporter(endpoint.String(), "test", "passw")

	xunit := &PolarionXUnit{
		Properties: []formatter.JUnitProperty{
			{Name: "polarion-project-id", Value: "TestProject"},
			{Name: "polarion-testrun-title", Value: "Test Title"},
		},
		Suites: []PolarionXUnitTestSuite{
			{
				JUnitTestSuite: formatter.JUnitTestSuite{
					Tests:    2,
					Failures: 1,
				},
				TestCases: []PolarionXUnitTestCase{
					{
						JUnitTestCase: formatter.JUnitTestCase{Name: "Success test case"},
						Properties:    []formatter.JUnitProperty{{Name: "polarion-testcase-id", Value: "Z01"}},
					},
					{
						JUnitTestCase: formatter.JUnitTestCase{
							Name: "Failed test case",
							Failure: &formatter.JUnitFailure{
								Message:  "Failed",
								Contents: "Some error",
							},
						},
						Properties: []formatter.JUnitProperty{{Name: "polarion-testcase-id", Value: "Z02"}},
					},
				},
			},
		},
	}

	jobID, err := x.Import(xunit)
	if err != nil {
		t.Fatalf("failed to import xunit with error: %s", err)
	}

	if expected := 99; jobID != expected {
		t.Fatalf("got jobID '%d' is not equal to the expected id '%d'", jobID, expected)
	}
}

func TestXUnitImporterGetJobStatus(t *testing.T) {

	endpoint, mux, teardown := setup()
	defer teardown()

	mux.HandleFunc(JobQueueEndpoint, func(w http.ResponseWriter, r *http.Request) {

		got := r.URL.Query().Get("jobIds")
		if expected := "99"; got != expected {
			t.Fatalf("got jobID '%s' but expected '%s'", got, expected)
		}

		fmt.Fprint(w, `{ "jobs": [{"status": "READY"}] }`)
	})

	x := NewXUnitImporter(endpoint.String(), "test", "passw")

	status, err := x.GetJobStatus(99)
	if err != nil {
		t.Fatalf("failed to import xunit with error: %s", err)
	}

	if expected := ReadyStatus; status != expected {
		t.Fatalf("got status '%s' but expected '%s'", status, expected)
	}

}
