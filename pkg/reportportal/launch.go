package reportportal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
)

const (
	importFileFieldName = "file"
)

type RPLaunchService service

func (s *RPLaunchService) Import(ctx context.Context, projectName string, importFile string, launchName string) (*RPLaunchResponse, error) {
	file, err := os.Open(importFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileName := fi.Name()
	if launchName != "" {
		fileName = launchName + ".zip"
	}
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(importFileFieldName, fileName)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}
	writer.WriteField("projectName", projectName)
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/launch/import", projectName)
	req, err := s.client.NewRequest("POST", u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
	launchResp := &RPLaunchResponse{}
	_, err = s.client.Do(ctx, req, launchResp)
	if err != nil {
		return nil, err
	}
	return launchResp, nil
}

func (s *RPLaunchService) Update(ctx context.Context, projectName string, launchId int, input *RPLaunchUpdateInput) (*RPLaunchResponse, error) {
	u := fmt.Sprintf("%s/launch/%d/update", projectName, launchId)
	req, err := s.client.NewRequest("PUT", u, input)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	launchResp := &RPLaunchResponse{}
	_, err = s.client.Do(ctx, req, launchResp)
	if err != nil {
		return nil, err
	}
	return launchResp, nil
}

func (s *RPLaunchService) Get(ctx context.Context, projectName string, launchUuid string) (*RPLaunchDetailsResponse, error) {
	u := fmt.Sprintf("%s/launch/uuid/%s", projectName, launchUuid)
	req, err := s.client.NewRequest("GET", u, new(bytes.Buffer))
	if err != nil {
		return nil, err
	}
	launchResp := &RPLaunchDetailsResponse{}
	_, err = s.client.Do(ctx, req, launchResp)
	if err != nil {
		return nil, err
	}
	return launchResp, nil
}
