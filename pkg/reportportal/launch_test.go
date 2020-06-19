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
		fmt.Fprint(w, `{"msg":"Launch with id = 5ef0edf5a2fd760001fe5a1c is successfully imported"}`)
	})

	f, _ := filepath.Abs("./testdata/test-archive.zip")
	resp, err := client.Launches.Import(context.TODO(), projectname, f, "testlaunch")
	if err != nil {
		t.Errorf("unexpected error when upload file: %v", err)
	}
	if resp.GetLaunchId() != "5ef0edf5a2fd760001fe5a1c" {
		t.Errorf("expected launch id: %s but got: %s", "5ef0edf5a2fd760001fe5a1c", resp.GetLaunchId())
	}
}

func TestRPLaunchService_Update(t *testing.T) {
	var projectname = "testproject"
	var launchId = "testlaunchid"
	client, mux, _, teardown := setup()
	defer teardown()

	input := &RPLaunchUpdateInput{
		Description: "update test",
		Tags:        []string{"test"},
	}

	mux.HandleFunc(fmt.Sprintf("/%s/launch/%s/update", projectname, launchId), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		b := &RPLaunchUpdateInput{}
		err := json.NewDecoder(r.Body).Decode(b)
		if err != nil {
			t.Errorf("can not parse request data due to error: %v", err)
		}
		if !reflect.DeepEqual(input, b) {
			t.Errorf("launch.Update input: %+v, got: %+v", input, b)
		}

		fmt.Fprint(w, `{"msg":"Launch with id = '5ef0edf5a2fd760001fe5a1c' is successfully updated"}`)
	})

	resp, err := client.Launches.Update(context.TODO(), projectname, launchId, input)
	if err != nil {
		t.Errorf("unexpected error when update launch: %v", err)
	}
	if resp.GetLaunchId() != "5ef0edf5a2fd760001fe5a1c" {
		t.Errorf("expected launch id %s, but got %s", "5ef0edf5a2fd760001fe5a1c", resp.GetLaunchId())
	}
}
