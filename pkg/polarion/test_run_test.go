package polarion

import (
	"fmt"
	"net/http"
	"testing"
)

func TestGetTestRunByID(t *testing.T) {
	cases := []struct {
		description string
		test        func(t *testing.T)
	}{{
		description: "should retrieve the test run",
		test: func(t *testing.T) {
			polarion, mux, teardown, _ := setupSession()
			defer teardown()

			mux.HandleFunc("/"+string(testManagementService), func(w http.ResponseWriter, r *http.Request) {
				testRequest(t, r, `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"><soapenv:Header xmlns:ses="http://ws.polarion.com/session"><ses:sessionID>fakesessionid</ses:sessionID></soapenv:Header><soapenv:Body xmlns:tes="http://ws.polarion.com/TestManagementWebService-impl"><tes:getTestRunById><tes:projectId>projectid</tes:projectId><tes:id>testrunid</tes:id></tes:getTestRunById></soapenv:Body></soapenv:Envelope>`)
				fmt.Fprint(w, `<Envelope><Body><getTestRunByIdResponse><getTestRunByIdReturn><id>testrunid</id></getTestRunByIdReturn></getTestRunByIdResponse></Body></Envelope>`)
			})

			run, err := polarion.GetTestRunByID("projectid", "testrunid")
			if err != nil {
				t.Fatalf("GetTestRunByID failed with error: %s", err)
			}

			if expected := "testrunid"; run.ID != expected {
				t.Fatalf("the test run ID should be '%s' but got '%s'", expected, run.ID)
			}
		},
	}, {
		description: "should fail to retrieve the test run",
		test: func(t *testing.T) {
			polarion, mux, teardown, _ := setupSession()
			defer teardown()

			mux.HandleFunc("/"+string(testManagementService), func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `<Envelope><Body><Fault><faultcode>some error</faultcode></Fault></Body></Envelope>`)
			})

			_, err := polarion.GetTestRunByID("projectid", "testrunid")
			if err == nil {
				t.Fatalf("GetTestRunByID should fails with error")
			}
		},
	}}

	for _, c := range cases {
		t.Run(c.description, c.test)
	}
}

func TestCreateTestRun(t *testing.T) {
	cases := []struct {
		description string
		test        func(t *testing.T)
	}{{
		description: "should create a test run",
		test: func(t *testing.T) {
			polarion, mux, teardown, _ := setupSession()
			defer teardown()

			mux.HandleFunc("/"+string(testManagementService), func(w http.ResponseWriter, r *http.Request) {
				testRequest(t, r, `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"><soapenv:Header xmlns:ses="http://ws.polarion.com/session"><ses:sessionID>fakesessionid</ses:sessionID></soapenv:Header><soapenv:Body xmlns:tes="http://ws.polarion.com/TestManagementWebService-impl"><tes:createTestRun><tes:project>projectid</tes:project><tes:id>testrunid</tes:id><tes:template>templateid</tes:template></tes:createTestRun></soapenv:Body></soapenv:Envelope>`)
				fmt.Fprint(w, `<Envelope><Body><createTestRunResponse><createTestRunReturn>testrunuri</createTestRunReturn></createTestRunResponse></Body></Envelope>`)
			})

			uri, err := polarion.CreateTestRun("projectid", "testrunid", "templateid")
			if err != nil {
				t.Fatalf("CreateTestRun failed with error: %s", err)
			}

			if expected := "testrunuri"; uri != expected {
				t.Fatalf("the URI should be '%s' but got '%s'", expected, uri)
			}
		},
	}, {
		description: "should fail to create the test run",
		test: func(t *testing.T) {
			polarion, mux, teardown, _ := setupSession()
			defer teardown()

			mux.HandleFunc("/"+string(testManagementService), func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `<Envelope><Body><Fault><faultcode>some error</faultcode></Fault></Body></Envelope>`)
			})

			_, err := polarion.CreateTestRun("projectid", "testrunid", "templateid")
			if err == nil {
				t.Fatalf("CreateTestRun should fails with error")
			}
		},
	}}

	for _, c := range cases {
		t.Run(c.description, c.test)
	}
}

func TestUpdateTestRun(t *testing.T) {
	cases := []struct {
		description string
		test        func(t *testing.T)
	}{{
		description: "should update a test run",
		test: func(t *testing.T) {
			polarion, mux, teardown, _ := setupSession()
			defer teardown()

			mux.HandleFunc("/"+string(testManagementService), func(w http.ResponseWriter, r *http.Request) {
				testRequest(t, r, `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"><soapenv:Header xmlns:ses="http://ws.polarion.com/session"><ses:sessionID>fakesessionid</ses:sessionID></soapenv:Header><soapenv:Body xmlns:tes="http://ws.polarion.com/TestManagementWebService-impl" xmlns:tes1="http://ws.polarion.com/TestManagementWebService-types" xmlns:trac="http://ws.polarion.com/TrackerWebService-types"><tes:updateTestRun><tes:content uri="testrunuri"><tes1:title>Test Run</tes1:title><tes1:isTemplate>true</tes1:isTemplate><tes1:customFields><trac:Custom><trac:key>plannedin</trac:key><trac:value><trac:id>3_8_2_rc1</trac:id></trac:value></trac:Custom></tes1:customFields></tes:content></tes:updateTestRun></soapenv:Body></soapenv:Envelope>`)
				fmt.Fprint(w, `<Envelope></Envelope>`)
			})

			err := polarion.UpdateTestRun("testrunuri", "Test Run", true, "3_8_2_rc1")
			if err != nil {
				t.Fatalf("UpdateTestRun failed with error: %s", err)
			}

		},
	}, {
		description: "should fail to update the test run",
		test: func(t *testing.T) {
			polarion, mux, teardown, _ := setupSession()
			defer teardown()

			mux.HandleFunc("/"+string(testManagementService), func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `<Envelope><Body><Fault><faultcode>some error</faultcode></Fault></Body></Envelope>`)
			})

			err := polarion.UpdateTestRun("testrunuri", "Test Run", true, "3_8_2_rc1")
			if err == nil {
				t.Fatalf("UpdateTestRun should fails with error")
			}
		},
	}}

	for _, c := range cases {
		t.Run(c.description, c.test)
	}

}
