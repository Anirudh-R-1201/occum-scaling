package occum

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"

	jamiethompsonmev1alpha1 "github.com/jthomperoo/predictive-horizontal-pod-autoscaler/api/v1alpha1"
)

const (
	defaultTimeout = 30000
)

const algorithmPath = "algorithms/prediction_adjustment/predict.py"

type occumParameters struct {
	LookAhead          int           `json:"lookAhead"`
	CurrentReplicas    int32         `json:"currentReplicas"`
	HistoricalReplicas []replicaData `json:"historicalReplicas"`
	CurrentTime        string        `json:"currentTime,omitempty"`
}

type replicaData struct {
	Time     string `json:"time"`
	Replicas int32  `json:"replicas"`
}

// Config represents an Occum prediction model configuration
type Config struct {
	StoredValues int `yaml:"storedValues"`
	LookAhead    int `yaml:"lookAhead"`
}

// Runner defines an algorithm runner, allowing algorithms to be run
type AlgorithmRunner interface {
	RunAlgorithmWithValue(algorithmPath string, value string, timeout int) (string, error)
}

// Predict provides logic for using Occum to make a prediction
type Predict struct {
	Runner AlgorithmRunner
}

// GetPrediction uses Occum to predict what the replica count should be
func (p *Predict) GetPrediction(model *jamiethompsonmev1alpha1.Model, history *jamiethompsonmev1alpha1.ModelHistory) (int32, error) {
	if model.Occum == nil {
		return 0, errors.New("no Occum configuration provided for model")
	}

	if history.TrafficHistory == nil || len(history.TrafficHistory) == 0 {
		return 0, errors.New("no traffic history provided for Occum model")
	}

	// Convert traffic history to replica history
	historicalReplicas := make([]replicaData, len(history.TrafficHistory))
	for i, th := range history.TrafficHistory {
		historicalReplicas[i] = replicaData{
			Time:     th.Time.Format("2006-01-02T15:04:05Z"),
			Replicas: int32(th.Traffic), // Using traffic value directly as replica count
		}
	}

	currentReplicas := int32(history.TrafficHistory[len(history.TrafficHistory)-1].Traffic)

	parameters, err := json.Marshal(occumParameters{
		LookAhead:          model.Occum.LookAhead,
		CurrentReplicas:    currentReplicas,
		HistoricalReplicas: historicalReplicas,
		CurrentTime:        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	})
	if err != nil {
		return 0, err
	}

	timeout := defaultTimeout
	if model.CalculationTimeout != nil {
		timeout = *model.CalculationTimeout
	}

	value, err := p.Runner.RunAlgorithmWithValue(algorithmPath, string(parameters), timeout)
	if err != nil {
		return 0, err
	}

	predictedReplicas, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	return int32(predictedReplicas), nil
}

func (p *Predict) PruneHistory(model *jamiethompsonmev1alpha1.Model, history *jamiethompsonmev1alpha1.ModelHistory) error {
	if model.Occum == nil {
		return errors.New("no Occum configuration provided for model")
	}

	if history.TrafficHistory == nil || len(history.TrafficHistory) <= model.Occum.HistorySize {
		return nil
	}

	// Sort by date created, newest first
	sort.Slice(history.TrafficHistory, func(i, j int) bool {
		return !history.TrafficHistory[i].Time.Before(history.TrafficHistory[j].Time)
	})

	// Keep only the newest entries up to HistorySize
	history.TrafficHistory = history.TrafficHistory[:model.Occum.HistorySize]
	return nil
}

// GetType returns the type of the Prediction model
func (p *Predict) GetType() string {
	return jamiethompsonmev1alpha1.TypeOccum
}
