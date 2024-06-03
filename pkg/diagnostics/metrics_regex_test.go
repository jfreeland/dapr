package diagnostics

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/dapr/dapr/pkg/config"
	diagUtils "github.com/dapr/dapr/pkg/diagnostics/utils"
)

const (
	componentLatencyConfigurationName     = "component/configuration/latencies"
	componentLatencyCryptoName            = "component/crypto/latencies"
	componentLatencyInputBindingName      = "component/input_binding/latencies"
	componentLatencyOutputBindingName     = "component/output_binding/latencies"
	componentLatencyPubsubEgressBulkName  = "component/pubsub_egress/bulk/latencies"
	componentLatencyPubsubEgressName      = "component/pubsub_egress/latencies"
	componentLatencyPubsubIngressBulkName = "component/pubsub_ingress/bulk/latencies"
	componentLatencyPubsubIngressName     = "component/pubsub_ingress/latencies"
	componentLatencySecretName            = "component/secret/latencies"
	componentLatencyStateName             = "component/state/latencies"
	grpcHealthprobesLatencyName           = "grpc.io/healthprobes/roundtrip_latency"
	grpcRoundtripLatencyName              = "grpc.io/client/roundtrip_latency"
	grpcServerLatencyName                 = "grpc.io/server/server_latency"
	httpClientRoundtripLatencyName        = "http/client/roundtrip_latency"
	httpHealthprobesLatencyName           = "http/healthprobes/roundtrip_latency"
	httpServerLatencyName                 = "http/server/latency"
	resiliencyActivationViewName          = "resiliency/activations_total"
	resiliencyCountViewName               = "resiliency/count"
	resiliencyLoadedViewName              = "resiliency/loaded"
	serviceInvocationRecvLatencyName      = "runtime/service_invocation/res_recv_latency_ms"
	workflowActivityLatencyName           = "runtime/workflow/activity/execution/latency"
	workflowExecutionLatencyName          = "runtime/workflow/execution/latency"
	workflowOperationLatencyName          = "runtime/workflow/operation/latency"
	workflowSchedulingLatencyName         = "runtime/workflow/scheduling/latency"
)

func cleanupRegisteredViews() {
	CleanupRegisteredViews(
		componentLatencyConfigurationName,
		componentLatencyCryptoName,
		componentLatencyInputBindingName,
		componentLatencyOutputBindingName,
		componentLatencyPubsubEgressBulkName,
		componentLatencyPubsubEgressName,
		componentLatencyPubsubIngressBulkName,
		componentLatencyPubsubIngressName,
		componentLatencySecretName,
		componentLatencyStateName,
		grpcHealthprobesLatencyName,
		grpcRoundtripLatencyName,
		grpcServerLatencyName,
		httpClientRoundtripLatencyName,
		httpHealthprobesLatencyName,
		httpServerLatencyName,
		resiliencyActivationViewName,
		resiliencyCountViewName,
		resiliencyLoadedViewName,
		serviceInvocationRecvLatencyName,
		workflowActivityLatencyName,
		workflowExecutionLatencyName,
		workflowOperationLatencyName,
		workflowSchedulingLatencyName,
	)
}

func TestRegexRulesSingle(t *testing.T) {
	const statName = "test_stat_regex"
	methodKey := tag.MustNewKey("method")
	testStat := stats.Int64(statName, "Stat used in unit test", stats.UnitDimensionless)
	latencyDistributionBuckets := []float64{5, 50, 500, 5_000}
	latencyDistribution := view.Distribution(latencyDistributionBuckets...)

	cleanupRegisteredViews()

	err := InitMetrics("testAppId2", "", []config.MetricsRule{
		{
			Name: statName,
			Labels: []config.MetricLabel{
				{
					Name: methodKey.Name(),
					Regex: map[string]string{
						"/orders/TEST":      "/orders/.+",
						"/lightsabers/TEST": "/lightsabers/.+",
					},
				},
			},
		},
	}, false, latencyDistributionBuckets)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("single regex rule applied", func(t *testing.T) {
		view.Register(
			diagUtils.NewMeasureView(testStat, []tag.Key{methodKey}, defaultSizeDistribution),
		)
		t.Cleanup(func() {
			view.Unregister(view.Find(statName))
		})

		spew.Dump(diagUtils.WithTags(testStat.Name(), methodKey, "/orders/456"))
		stats.RecordWithTags(context.Background(),
			diagUtils.WithTags(testStat.Name(), methodKey, "/orders/123"),
			testStat.M(1))

		viewData, _ := view.RetrieveData(statName)
		spew.Dump(viewData)
		v := view.Find(statName)

		allTagsPresent(t, v, viewData[0].Tags)

		assert.Equal(t, "/orders/TEST", viewData[0].Tags[0].Value)
	})

	t.Run("single regex rule not applied", func(t *testing.T) {
		view.Register(
			diagUtils.NewMeasureView(testStat, []tag.Key{methodKey}, defaultSizeDistribution),
		)
		t.Cleanup(func() {
			view.Unregister(view.Find(statName))
		})

		s := newGRPCMetrics()
		s.Init("test", latencyDistribution)

		stats.RecordWithTags(context.Background(),
			diagUtils.WithTags(testStat.Name(), methodKey, "/siths/123"),
			testStat.M(1))

		viewData, _ := view.RetrieveData(statName)
		v := view.Find(statName)

		allTagsPresent(t, v, viewData[0].Tags)

		assert.Equal(t, "/siths/123", viewData[0].Tags[0].Value)
	})

	t.Run("correct regex rules applied", func(t *testing.T) {
		view.Register(
			diagUtils.NewMeasureView(testStat, []tag.Key{methodKey}, defaultSizeDistribution),
		)
		t.Cleanup(func() {
			view.Unregister(view.Find(statName))
		})

		s := newGRPCMetrics()
		s.Init("test", latencyDistribution)

		stats.RecordWithTags(context.Background(),
			diagUtils.WithTags(testStat.Name(), methodKey, "/orders/123"),
			testStat.M(1))
		stats.RecordWithTags(context.Background(),
			diagUtils.WithTags(testStat.Name(), methodKey, "/lightsabers/123"),
			testStat.M(1))

		viewData, _ := view.RetrieveData(statName)

		orders := false
		lightsabers := false

		for _, v := range viewData {
			if v.Tags[0].Value == "/orders/TEST" {
				orders = true
			} else if v.Tags[0].Value == "/lightsabers/TEST" {
				lightsabers = true
			}
		}

		assert.True(t, orders)
		assert.True(t, lightsabers)
	})
}
