package reportportal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRPLaunchService_Import(t *testing.T) {
	var projectname = "testproject"
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc(fmt.Sprintf("/%s/launch/import", projectname), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		file, _, err := r.FormFile(importFileFieldName)
		if err != nil {
			t.Errorf("error parsing uploaded file: %v", err)
		}
		if file == nil {
			t.Error("no file found in the import request")
		}
		fmt.Fprint(w, `{"message":"Launch with id = b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3 is successfully imported."}`)
	})

	f, _ := filepath.Abs("./testdata/test-archive.zip")
	resp, err := client.Launches.Import(context.TODO(), projectname, f, "testlaunch")
	if err != nil {
		t.Errorf("unexpected error when upload file: %v", err)
	}
	if resp.GetLaunchUuid() != "b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3" {
		t.Errorf("expected launch id: %s but got: %s", "b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3", resp.GetLaunchUuid())
	}
}

func TestRPLaunchService_Update(t *testing.T) {
	var projectname = "testproject"
	var launchId = 1234
	var launchUuid = "b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3"
	client, mux, _, teardown := setup()
	defer teardown()

	input := &RPLaunchUpdateInput{
		Description: "update test",
		Tags:        []string{"test"},
	}

	mux.HandleFunc(fmt.Sprintf("/%s/launch/%d/update", projectname, launchId), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		b := &RPLaunchUpdateInput{}
		err := json.NewDecoder(r.Body).Decode(b)
		if err != nil {
			t.Errorf("can not parse request data due to error: %v", err)
		}
		if !reflect.DeepEqual(input, b) {
			t.Errorf("launch.Update input: %+v, got: %+v", input, b)
		}

		fmt.Fprintf(w, `{"message":"Launch with id = '%s' is successfully updated"}`, launchUuid)
	})

	resp, err := client.Launches.Update(context.TODO(), projectname, launchId, input)
	if err != nil {
		t.Errorf("unexpected error when update launch: %v", err)
	}
	if resp.GetLaunchUuid() != launchUuid {
		t.Errorf("expected launch id %s, but got %s", launchUuid, resp.GetLaunchUuid())
	}
}

func TestRPLaunchService_Get(t *testing.T) {
	var projectname = "testproject"
	var launchId = 1234
	var launchUuid = "b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3"
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc(fmt.Sprintf("/%s/launch/uuid/%s", projectname, launchUuid), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		fmt.Fprintf(w, `{"id":%d}`, launchId)
	})

	resp, err := client.Launches.Get(context.TODO(), projectname, launchUuid)
	if err != nil {
		t.Errorf("unexpected error when getting launch with UUID %s: %v", launchUuid, err)
	}
	if resp.Id != launchId {
		t.Errorf("expected launch id %d, but got %d", launchId, resp.Id)
	}
}
