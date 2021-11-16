package reportportal

import (
	"context"
	"regexp"
)

type RPLaunchResponse struct {
	Message string `json:"message"`
}

type RPLaunchDetailsResponse struct {
	Id int `json:"id"`
}

func (r *RPLaunchResponse) GetLaunchUuid() string {
	// the msg field is something like "Launch with id = b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3 is successfully imported."
	// or "Launch with id = b862b3c3-a9ce-47d1-9f5c-e51ae9de50f3 is successfully updated."
	// need a regexp to get the id value
	var re = regexp.MustCompile(`.*?\=\s?'?([a-zA-Z0-9-]*?)'?\s`)
	m := re.FindStringSubmatch(r.Message)
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
	Update(ctx context.Context, projectName string, launchId int, input *RPLaunchUpdateInput) (*RPLaunchResponse, error)
	Get(ctx context.Context, projectName string, launchUuid string) (*RPLaunchDetailsResponse, error)
}
