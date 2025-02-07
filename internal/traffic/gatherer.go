package traffic

import (
	jamiethompsonmev1alpha1 "github.com/jthomperoo/predictive-horizontal-pod-autoscaler/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Gatherer defines methods for gathering traffic metrics
type Gatherer interface {
	GatherTraffic(phpa *jamiethompsonmev1alpha1.PredictiveHorizontalPodAutoscaler) (float64, error)
} 