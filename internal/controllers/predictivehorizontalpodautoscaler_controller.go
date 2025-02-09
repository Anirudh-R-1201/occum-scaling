package controllers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/anirudh-r-1201/occum-scaling/internal/traffic"
	"github.com/jthomperoo/k8shorizmetrics/v2"
	"github.com/jthomperoo/predictive-horizontal-pod-autoscaler/api/v1alpha1"
	"github.com/jthomperoo/predictive-horizontal-pod-autoscaler/internal/prediction"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PredictiveHorizontalPodAutoscalerReconciler reconciles a PredictiveHorizontalPodAutoscaler object
type PredictiveHorizontalPodAutoscalerReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	ScaleClient scale.ScalesGetter
	Gatherer    k8shorizmetrics.Gatherer
	Evaluator   k8shorizmetrics.Evaluator
	Predicter   prediction.Predicter
	Traffic     traffic.Gatherer
}

func (r *PredictiveHorizontalPodAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	phpa := &v1alpha1.PredictiveHorizontalPodAutoscaler{}
	if err := r.Get(ctx, req.NamespacedName, phpa); err != nil {
		if errors.IsNotFound(err) {
			log.Info("PredictiveHorizontalPodAutoscaler not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Error getting PredictiveHorizontalPodAutoscaler")
		return ctrl.Result{}, err
	}

	// For Occum models, gather and store traffic metrics
	for _, model := range phpa.Spec.Models {
		if model.Occum != nil {
			// Gather current traffic
			currentTraffic, err := r.Traffic.GatherTraffic(phpa)
			if err != nil {
				log.Error(err, "failed to gather traffic metrics")
				return ctrl.Result{}, err
			}

			// Find or create model history
			var modelHistory *v1alpha1.ModelHistory
			for i := range phpa.Status.ModelHistories {
				if phpa.Status.ModelHistories[i].Type == v1alpha1.TypeOccum {
					modelHistory = &phpa.Status.ModelHistories[i]
					break
				}
			}

			if modelHistory == nil {
				modelHistory = &v1alpha1.ModelHistory{
					Type:           v1alpha1.TypeOccum,
					TrafficHistory: []v1alpha1.TimestampedTraffic{},
				}
				phpa.Status.ModelHistories = append(phpa.Status.ModelHistories, *modelHistory)
			}

			// Add new traffic measurement
			now := metav1.Now()
			modelHistory.TrafficHistory = append(modelHistory.TrafficHistory, v1alpha1.TimestampedTraffic{
				Time:    &now,
				Traffic: currentTraffic,
			})

			// Prepare input for prediction algorithm
			algorithmInput := struct {
				LookAhead         int                           `json:"lookAhead"`
				CurrentReplicas   int32                         `json:"currentReplicas"`
				HistoricalTraffic []v1alpha1.TimestampedTraffic `json:"historicalTraffic"`
				CurrentTime       string                        `json:"currentTime"`
				MaxReplicas       *int32                        `json:"maxReplicas,omitempty"`
			}{
				LookAhead:         model.Occum.LookAhead,
				CurrentReplicas:   phpa.Status.CurrentReplicas,
				HistoricalTraffic: modelHistory.TrafficHistory,
				CurrentTime:       time.Now().UTC().Format(time.RFC3339),
				MaxReplicas:       phpa.Spec.MaxReplicas,
			}

			// Convert to JSON
			inputJSON, err := json.Marshal(algorithmInput)
			if err != nil {
				log.Error(err, "failed to marshal algorithm input")
				return ctrl.Result{}, err
			}

			// TODO: Call prediction algorithm with inputJSON
		}
	}

	return ctrl.Result{}, nil
}
