package services

import (
	"context"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"time"
)

type PrometheusService interface {
	Query(ctx context.Context, query string, ts time.Time) (model.Value, api.Warnings, error)
	QueryRange(ctx context.Context, query string, r promv1.Range) (model.Value, api.Warnings, error)
}
