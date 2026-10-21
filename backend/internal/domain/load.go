package domain

import (
	_ "embed"
	"encoding/json"
)

//go:embed load_thresholds.json
var thresholdsJSON []byte

// Thresholds are the workload band limits as fractions of the team average.
// The same file is imported by the frontend.
type Thresholds struct {
	OverloadedAbove  float64 `json:"overloadedAbove"`
	UnderloadedBelow float64 `json:"underloadedBelow"`
}

// LoadThresholds is the parsed load_thresholds.json.
var LoadThresholds = mustThresholds()

func mustThresholds() Thresholds {
	var t Thresholds
	if err := json.Unmarshal(thresholdsJSON, &t); err != nil {
		panic(err)
	}
	return t
}

// Load statuses.
const (
	LoadBalanced    = "Balanced"
	LoadOverloaded  = "Overloaded"
	LoadUnderloaded = "Underloaded"
)

// TeamAverage is the total active load divided by the number of members
// (members with 0 h count).
func TeamAverage(active []float64) float64 {
	if len(active) == 0 {
		return 0
	}
	var sum float64
	for _, a := range active {
		sum += a
	}
	return sum / float64(len(active))
}

// LoadStatus classifies one member's active hours against the team average.
// Overloaded is strictly above the upper threshold; exactly at it is Balanced.
func LoadStatus(active, average float64) string {
	if average <= 0 {
		return LoadBalanced
	}
	// eps absorbs float noise so exactly 130% stays Balanced.
	const eps = 1e-9
	switch {
	case active > average*LoadThresholds.OverloadedAbove+eps:
		return LoadOverloaded
	case active < average*LoadThresholds.UnderloadedBelow-eps:
		return LoadUnderloaded
	}
	return LoadBalanced
}
