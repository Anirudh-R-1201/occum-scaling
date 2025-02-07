package prometheus

import (
	"context"
	"fmt"
	"time"

	jamiethompsonmev1alpha1 "github.com/jthomperoo/predictive-horizontal-pod-autoscaler/api/v1alpha1"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Gatherer gathers traffic metrics from Prometheus
type Gatherer struct {
	client v1.API
}

// NewGatherer creates a new Prometheus traffic gatherer
func NewGatherer(address string) (*Gatherer, error) {
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus client: %w", err)
	}

	return &Gatherer{
		client: v1.NewAPI(client),
	}, nil
}

// GatherTraffic gets the current traffic value from Prometheus
func (g *Gatherer) GatherTraffic(phpa *jamiethompsonmev1alpha1.PredictiveHorizontalPodAutoscaler) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Assuming the traffic query is stored in the PHPA spec
	if phpa.Spec.TrafficMetric == nil || phpa.Spec.TrafficMetric.PrometheusQuery == "" {
		return 0, fmt.Errorf("no prometheus query configured for traffic metric")
	}

	result, warnings, err := g.client.Query(ctx, phpa.Spec.TrafficMetric.PrometheusQuery, time.Now())
	if err != nil {
		return 0, fmt.Errorf("error querying prometheus: %w", err)
	}

	if len(warnings) > 0 {
		// Log warnings but continue
		fmt.Printf("Warnings from Prometheus query: %v\n", warnings)
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return 0, fmt.Errorf("unexpected result format from prometheus")
	}

	if len(vector) == 0 {
		return 0, fmt.Errorf("no data points returned from prometheus")
	}

	// Use the first value if multiple series are returned
	return float64(vector[0].Value), nil
}