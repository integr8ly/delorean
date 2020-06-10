package polarion

import (
	"fmt"
	"net/http"
	"testing"
)

func TestGetPlanByID(t *testing.T) {
	cases := []struct {
		description string
		test        func(t *testing.T)
	}{{
		description: "should retrieve the plan",
		test: func(t *testing.T) {
			polarion, mux, teardown, _ := setupSession()
			defer teardown()

			mux.HandleFunc("/"+string(planningService), func(w http.ResponseWriter, r *http.Request) {
				testRequest(t, r, `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"><soapenv:Header xmlns:ses="http://ws.polarion.com/session"><ses:sessionID>fakesessionid</ses:sessionID></soapenv:Header><soapenv:Body xmlns:plan="http://ws.polarion.com/PlanningWebService-impl"><plan:getPlanById><plan:projectId>projectid</plan:projectId><plan:id>planid</plan:id></plan:getPlanById></soapenv:Body></soapenv:Envelope>`)
				fmt.Fprint(w, `<Envelope><Body><getPlanByIdResponse><getPlanByIdReturn><id>planid</id></getPlanByIdReturn></getPlanByIdResponse></Body></Envelope>`)
			})

			plan, err := polarion.GetPlanByID("projectid", "planid")
			if err != nil {
				t.Fatalf("GetPlanByID failed with error: %s", err)
			}

			if expected := "planid"; plan.ID != expected {
				t.Fatalf("the plan ID should be '%s' but got '%s'", expected, plan.ID)
			}
		},
	}, {
		description: "should fail to retrieve the plan",
		test: func(t *testing.T) {
			polarion, mux, teardown, _ := setupSession()
			defer teardown()

			mux.HandleFunc("/"+string(planningService), func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `<Envelope><Body><Fault><faultcode>some error</faultcode></Fault></Body></Envelope>`)
			})

			_, err := polarion.GetPlanByID("projectid", "planid")
			if err == nil {
				t.Fatalf("GetPlanByID should fails with error")
			}
		},
	},
	}

	for _, c := range cases {
		t.Run(c.description, c.test)
	}
}

func TestCreatePlan(t *testing.T) {
	cases := []struct {
		description string
		test        func(t *testing.T)
	}{{
		description: "should create the plan",
		test: func(t *testing.T) {
			polarion, mux, teardown, _ := setupSession()
			defer teardown()

			mux.HandleFunc("/"+string(planningService), func(w http.ResponseWriter, r *http.Request) {
				testRequest(t, r, `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"><soapenv:Header xmlns:ses="http://ws.polarion.com/session"><ses:sessionID>fakesessionid</ses:sessionID></soapenv:Header><soapenv:Body xmlns:plan="http://ws.polarion.com/PlanningWebService-impl"><plan:createPlan><plan:projectId>projectid</plan:projectId><plan:name>Plan Name</plan:name><plan:id>planid</plan:id><plan:parentId>parentid</plan:parentId><plan:templateId>templateid</plan:templateId></plan:createPlan></soapenv:Body></soapenv:Envelope>`)
				fmt.Fprint(w, `<Envelope></Envelope>`)
			})

			err := polarion.CreatePlan("projectid", "Plan Name", "planid", "parentid", "templateid")
			if err != nil {
				t.Fatalf("GetPlanByID failed with error: %s", err)
			}
		},
	}, {
		description: "should fail to create the plan",
		test: func(t *testing.T) {
			polarion, mux, teardown, _ := setupSession()
			defer teardown()

			mux.HandleFunc("/"+string(planningService), func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `<Envelope><Body><Fault><faultcode>some error</faultcode></Fault></Body></Envelope>`)
			})

			err := polarion.CreatePlan("projectid", "Plan Name", "planid", "parentid", "templateid")
			if err == nil {
				t.Fatalf("GetPlanByID should fails with error")
			}
		},
	},
	}

	for _, c := range cases {
		t.Run(c.description, c.test)
	}
}
