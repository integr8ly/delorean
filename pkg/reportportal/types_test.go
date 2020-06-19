package reportportal

import "testing"

func TestRPLaunchResponse_GetLaunchId(t *testing.T) {
	cases := []struct {
		description string
		msg         string
		expectedId  string
	}{
		{
			description: "success without quote",
			msg:         "Launch with id = 5ef0edf5a2fd760001fe5a1c is successfully imported",
			expectedId:  "5ef0edf5a2fd760001fe5a1c",
		},
		{
			description: "success with quote",
			msg:         "Launch with ID = '5ef0ea7da2fd760001fe59f3' successfully updated",
			expectedId:  "5ef0ea7da2fd760001fe59f3",
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			r := RPLaunchResponse{Msg: c.msg}
			actualId := r.GetLaunchId()
			if actualId != c.expectedId {
				t.Fatalf("expected id: %s got: %s", c.expectedId, actualId)
			}
		})
	}
}
