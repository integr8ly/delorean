package reportportal

import (
	"context"
	"regexp"
)

type RPLaunchResponse struct {
	Msg string `json:"msg"`
}

func (r *RPLaunchResponse) GetLaunchId() string {
	// the msg field is something like "Launch with id = 5ef0edf5a2fd760001fe5a1c is successfully imported."
	// or "Launch with ID = '5ef0ea7da2fd760001fe59f3' successfully updated."
	// need a regexp to get the id value
	var re = regexp.MustCompile(`.*?\=\s?'?([a-zA-Z0-9]*?)'?\s`)
	m := re.FindStringSubmatch(r.Msg)
	if m != nil {
		return m[1]
	}
	return ""
}

type RPLaunchUpdateInput struct {
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type RPLaunchManager interface {
	Import(ctx context.Context, projectName string, importFile string, launchName string) (*RPLaunchResponse, error)
	Update(ctx context.Context, projectName string, launchId string, input *RPLaunchUpdateInput) (*RPLaunchResponse, error)
}
