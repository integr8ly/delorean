package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	routeclientv1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"math"
	"path"
	"strconv"
	"strings"
	"time"
)

type queryType string

const (
	prometheusNamespace           = "redhat-rhmi-middleware-monitoring-operator"
	prometheusRouteName           = "prometheus-route"
	promQueryType       queryType = "query"
	promQueryRangeType  queryType = "query_range"
	defaultWorkers                = 5
	defaultQueryTimeout           = 10
)

type queryOpts struct {
	QueryType queryType `json:"type"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
}

type queryResult struct {
	v      model.Value
	Name   string          `json:"name"`
	Query  string          `json:"query"`
	Type   model.ValueType `json:"resultType"`
	Result json.RawMessage `json:"result`
}

func (qr *queryResult) MarshalJSON() ([]byte, error) {
	v := struct {
		Name   string          `json:"name"`
		Type   model.ValueType `json:"resultType"`
		Result model.Value     `json:"result"`
		Query  string          `json:"query"`
	}{
		Name:   qr.Name,
		Type:   qr.v.Type(),
		Query:  qr.Query,
		Result: qr.v,
	}
	return json.Marshal(v)
}

func (qr *queryResult) UnmarshalJSON(b []byte) error {
	v := struct {
		Name   string          `json:"name"`
		Type   model.ValueType `json:"resultType"`
		Result json.RawMessage `json:"result"`
		Query  string          `json:"query"`
	}{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	qr.Name = v.Name
	qr.Type = v.Type
	qr.Query = v.Query
	qr.Result = v.Result

	switch v.Type {
	case model.ValScalar:
		var sv model.Scalar
		err = json.Unmarshal(v.Result, &sv)
		qr.v = &sv

	case model.ValVector:
		var vv model.Vector
		err = json.Unmarshal(v.Result, &vv)
		qr.v = vv

	case model.ValMatrix:
		var mv model.Matrix
		err = json.Unmarshal(v.Result, &mv)
		qr.v = mv

	case model.ValString:
		var ms model.String
		err = json.Unmarshal(v.Result, &ms)
		qr.v = &ms

	default:
		err = fmt.Errorf("unexpected value type %q", v.Type)
	}
	return err
}

type queryResults struct {
	Name    string        `json:"name"`
	Results []queryResult `json:"results"`
}

type queryReportConfig struct {
	Name    string      `json:"name"`
	Queries []queryOpts `json:"queries"`
}

type queryRange struct {
	start    time.Time
	end      time.Time
	duration time.Duration
}

type queryReportCmd struct {
	namespace  string
	outputDir  string
	promAPI    services.PrometheusService
	timeout    int
	config     *queryReportConfig
	queryRange queryRange
}

type queryReportCmdFlags struct {
	namespace  string
	outputDir  string
	timeout    int
	configFile string
	start      int64
	end        int64
	duration   time.Duration
}

func init() {
	f := &queryReportCmdFlags{}
	cmd := &cobra.Command{
		Use:   "query-report",
		Short: "Run query against Prometheus on the target RHMI cluster and create reports",
		Run: func(cmd *cobra.Command, args []string) {
			kubeConfig, err := requireValue(KubeConfigKey)
			if err != nil {
				handleError(err)
			}
			c, err := newQueryReportCmd(kubeConfig, f)
			if err != nil {
				handleError(err)
			}
			if err = c.run(cmd.Context()); err != nil {
				handleError(err)
			}
		},
	}
	pipelineCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&f.outputDir, "output", "o", "", "Absolute path of the output directory to save reports")
	cmd.MarkFlagRequired("output")
	cmd.Flags().StringVarP(&f.namespace, "namespace", "n", prometheusNamespace, "The namespace to find the Prometheus route")
	cmd.Flags().StringVar(&f.configFile, "config-file", "", "Path to the query configuration file")
	cmd.MarkFlagRequired("config-file")
	cmd.Flags().IntVarP(&f.timeout, "timeout", "t", defaultQueryTimeout, "Timeout value for executing Prometheus queries")
	cmd.Flags().Int64Var(&f.end, "end-time", time.Now().Unix(), "End time for queryRange type of queries. Default to current time")
	cmd.Flags().Int64Var(&f.start, "start-time", 0, "Start time for queryRange type of queries. Only either start-time or duration should be specified")
	cmd.Flags().DurationVar(&f.duration, "duration", time.Duration(2*time.Hour), "Duration for queryRange type of queries. Only either start-time or duration should be specified")
}

func newQueryReportCmd(kubeconfig string, f *queryReportCmdFlags) (*queryReportCmd, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	routeclient, err := routeclientv1.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	promRoute, err := routeclient.Routes(f.namespace).Get(prometheusRouteName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	promUrl := fmt.Sprintf("https://%s", promRoute.Spec.Host)
	fmt.Println("Prometheus URL:", promUrl)
	promAPI, err := newPromAPI(promUrl, config.BearerToken)
	if err != nil {
		return nil, err
	}
	qrConfig := &queryReportConfig{}
	if err := utils.PopulateObjectFromYAML(f.configFile, qrConfig); err != nil {
		return nil, err
	}

	queryRange := parseQueryRange(f)

	return &queryReportCmd{
		namespace:  f.namespace,
		outputDir:  f.outputDir,
		promAPI:    promAPI,
		timeout:    f.timeout,
		config:     qrConfig,
		queryRange: queryRange,
	}, nil
}

func parseQueryRange(f *queryReportCmdFlags) queryRange {
	end := time.Unix(f.end, 0)
	var start time.Time
	var duration time.Duration
	// always use the start value if it's specified
	if f.start != 0 {
		start = time.Unix(f.start, 0)
		duration = end.Sub(start)
	} else {
		duration = f.duration
		start = end.Add(-duration)
	}
	queryRange := queryRange{
		start:    start,
		end:      end,
		duration: duration,
	}
	return queryRange
}

func (c *queryReportCmd) run(ctx context.Context) error {
	results, err := c.processQueries(ctx)
	if err != nil {
		return err
	}
	n := strings.ToLower(c.config.Name)
	n = strings.ReplaceAll(n, " ", "-")
	outputFile := path.Join(c.outputDir, fmt.Sprintf("%s.yaml", n))
	r := &queryResults{Name: c.config.Name, Results: results}
	if err := utils.WriteObjectToYAML(r, outputFile); err != nil {
		return err
	}
	fmt.Println("Report is generated:", outputFile)
	return nil
}

func (c *queryReportCmd) processQueries(ctx context.Context) ([]queryResult, error) {
	tasks := make([]utils.Task, len(c.config.Queries))
	for i, q := range c.config.Queries {
		v := q
		t := func() (utils.TaskResult, error) {
			r, err := c.queryProm(ctx, v)
			if err != nil {
				return nil, err
			}
			return queryResult{Name: v.Name, Query: v.Query, v: r}, nil
		}
		tasks[i] = t
	}
	results, err := utils.ParallelLimit(ctx, tasks, defaultWorkers)
	if err != nil {
		return nil, err
	}
	qr := make([]queryResult, len(results))
	for i, r := range results {
		qr[i] = r.(queryResult)
	}
	return qr, err
}

func (c *queryReportCmd) queryProm(ctx context.Context, opts queryOpts) (model.Value, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.timeout)*time.Second)
	defer cancel()
	switch opts.QueryType {
	case promQueryType:
		query := c.parseQueryRangeQuery(opts.Query)
		return c.promAPI.Query(ctx, query, time.Now())
	case promQueryRangeType:
		query := c.parseQueryRangeQuery(opts.Query)
		r := promv1.Range{Start: c.queryRange.start, End: c.queryRange.end, Step: 30 * time.Second}
		return c.promAPI.QueryRange(ctx, query, r)
	default:
		return nil, fmt.Errorf("unsupported query type: %s", opts.QueryType)
	}
}

func (c *queryReportCmd) parseQueryRangeQuery(query string) string {
	q := strings.ReplaceAll(query, "$range", strconv.FormatInt(c.queryRange.duration.Milliseconds(), 10))
	q = strings.ReplaceAll(q, "$duration", fmt.Sprintf("%ds", int64(math.Round(c.queryRange.duration.Seconds()))))
	return q
}

func newPromAPI(url string, token string) (promv1.API, error) {
	rt := config.NewBearerAuthRoundTripper(config.Secret(token), api.DefaultRoundTripper)
	client, err := api.NewClient(api.Config{Address: url, RoundTripper: rt})
	if err != nil {
		return nil, err
	}
	promAPI := promv1.NewAPI(client)
	return promAPI, nil
}
