package occum

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strconv"

	jamiethompsonmev1alpha1 "github.com/jthomperoo/predictive-horizontal-pod-autoscaler/api/v1alpha1"
)

const (
	defaultTimeout = 30000
)

const algorithmPath = "algorithms/prediction_adjustment/predict.py"

type occumParameters struct {
	LookAhead         int           `json:"lookAhead"`
	CurrentTraffic    float64       `json:"currentTraffic"`
	HistoricalTraffic []trafficData `json:"historicalTraffic"`
	CurrentTime       string        `json:"currentTime,omitempty"`
	TrafficPerReplica float64       `json:"trafficPerReplica"`
}

type trafficData struct {
	Time    string  `json:"time"`
	Traffic float64 `json:"traffic"`
}

// Config represents an Occum prediction model configuration
type Config struct {
	StoredValues      int     `yaml:"storedValues"`
	LookAhead         int     `yaml:"lookAhead"`
	TrafficPerReplica float64 `yaml:"trafficPerReplica"`
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

	// Convert traffic history to the format expected by the Python algorithm
	historicalTraffic := make([]trafficData, len(history.TrafficHistory))
	for i, th := range history.TrafficHistory {
		historicalTraffic[i] = trafficData{
			Time:    th.Time.Format("2006-01-02T15:04:05Z"),
			Traffic: th.Traffic,
		}
	}

	parameters, err := json.Marshal(occumParameters{
		LookAhead:         model.Occum.LookAhead,
		CurrentTraffic:    history.TrafficHistory[len(history.TrafficHistory)-1].Traffic,
		HistoricalTraffic: historicalTraffic,
		TrafficPerReplica: model.Occum.TrafficPerReplica,
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

	return int32(math.Ceil(predictedReplicas)), nil
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
