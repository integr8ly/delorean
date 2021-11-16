package reportportal

import "testing"

func TestRPLaunchResponse_GetLaunchUuid(t *testing.T) {
	cases := []struct {
		description  string
		msg          string
		expectedUuid string
	}{
		{
			description:  "success without quote",
			msg:          "Launch with id = b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3 is successfully imported",
			expectedUuid: "b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3",
		},
		{
			description:  "success with quote",
			msg:          "Launch with ID = 'b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3' successfully updated",
			expectedUuid: "b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3",
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			r := RPLaunchResponse{Message: c.msg}
			actualUuid := r.GetLaunchUuid()
			if actualUuid != c.expectedUuid {
				t.Fatalf("expected id: %s got: %s", c.expectedUuid, actualUuid)
			}
		})
	}
}
