package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

type mockPromService struct {
	queryFunc      func(ctx context.Context, query string, ts time.Time) (model.Value, error)
	queryRangeFunc func(ctx context.Context, query string, r promv1.Range) (model.Value, error)
}

func (m *mockPromService) Query(ctx context.Context, query string, ts time.Time) (model.Value, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, ts)
	}
	panic("not implemented")
}

func (m *mockPromService) QueryRange(ctx context.Context, query string, r promv1.Range) (model.Value, error) {
	if m.queryRangeFunc != nil {
		return m.queryRangeFunc(ctx, query, r)
	}
	panic("not implemented")
}

func TestQueryReportCmd(t *testing.T) {
	outputDir, err := ioutil.TempDir("/tmp", "query-report-test-")
	if err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	defer os.RemoveAll(outputDir)
	namespace := "test"
	queries := []queryOpts{
		{
			QueryType: promQueryType,
			Name:      "downtime_product_1",
			Query:     "downtime{product:1}",
		},
		{
			QueryType: promQueryType,
			Name:      "downtime_product_2",
			Query:     "downtime{product:2}",
		},
	}
	cases := []struct {
		description string
		promAPI     services.PrometheusService
		config      *queryReportConfig
		checkResult func(err error) error
	}{
		{
			description: "queries run successfully",
			promAPI: &mockPromService{
				queryFunc: func(ctx context.Context, query string, ts time.Time) (value model.Value, err error) {
					return &model.String{Value: query, Timestamp: model.Now()}, nil
				},
			},
			config: &queryReportConfig{
				Name:    "Test Report",
				Queries: queries,
			},
			checkResult: func(err error) error {
				if err != nil {
					return err
				}
				outputFile := path.Join(outputDir, "test-report.yaml")
				results := queryResults{}
				err = utils.PopulateObjectFromYAML(outputFile, &results)
				if err != nil {
					return err
				}
				if len(results.Results) != 2 {
					return fmt.Errorf("results should have %d items but got %d", 2, len(results.Results))
				}
				for i, r := range results.Results {
					t := r.v.(*model.String)
					if queries[i].Query != t.Value {
						return fmt.Errorf("expected: %s, got: %s", queries[i].Query, t.Value)
					}
				}
				return nil
			},
		},
		{
			description: "check query returns error",
			promAPI: &mockPromService{
				queryFunc: func(ctx context.Context, query string, ts time.Time) (value model.Value, err error) {
					return nil, errors.New("unexpected error")
				},
			},
			config: &queryReportConfig{
				Name:    "test",
				Queries: queries,
			},
			checkResult: func(err error) error {
				if err == nil {
					return errors.New("should have error but got nil")
				}
				return nil
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			cmd := &queryReportCmd{
				namespace: namespace,
				outputDir: outputDir,
				promAPI:   c.promAPI,
				config:    c.config,
			}
			err := cmd.run(context.TODO())
			if e := c.checkResult(err); e != nil {
				t.Fatalf("unexpected test error: %v", e)
			}
		})
	}
}

func TestQueryResult_UnmarshalJSON(t *testing.T) {
	r := "./testdata/queryReport/downtime-report.yaml"
	qr := &queryResults{}
	err := utils.PopulateObjectFromYAML(r, qr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, re := range qr.Results {
		if re.Result == nil {
			t.Fatal("result is nil")
		}
	}
}
