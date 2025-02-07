package controllers

import (
	"context"

	"github.com/jthomperoo/predictive-horizontal-pod-autoscaler/api/v1alpha1"
	"github.com/your-org/occum-scaling/internal/traffic"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

	phpa := &jamiethompsonmev1alpha1.PredictiveHorizontalPodAutoscaler{}
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
			var modelHistory *jamiethompsonmev1alpha1.ModelHistory
			for i := range phpa.Status.ModelHistories {
				if phpa.Status.ModelHistories[i].Type == jamiethompsonmev1alpha1.TypeOccum {
					modelHistory = &phpa.Status.ModelHistories[i]
					break
				}
			}

			if modelHistory == nil {
				modelHistory = &jamiethompsonmev1alpha1.ModelHistory{
					Type:          jamiethompsonmev1alpha1.TypeOccum,
					TrafficHistory: []jamiethompsonmev1alpha1.TimestampedTraffic{},
				}
				phpa.Status.ModelHistories = append(phpa.Status.ModelHistories, *modelHistory)
			}

			// Add new traffic measurement
			now := metav1.Now()
			modelHistory.TrafficHistory = append(modelHistory.TrafficHistory, jamiethompsonmev1alpha1.TimestampedTraffic{
				Time:    &now,
				Traffic: currentTraffic,
			})
		}
	}

	return ctrl.Result{}, nil
} 