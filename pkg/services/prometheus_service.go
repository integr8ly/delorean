package services

import (
	"context"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"time"
)

type PrometheusService interface {
	Query(ctx context.Context, query string, ts time.Time) (model.Value, error)
	QueryRange(ctx context.Context, query string, r promv1.Range) (model.Value, error)
}
