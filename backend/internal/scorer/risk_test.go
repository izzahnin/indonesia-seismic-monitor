package scorer

import (
	"testing"

	"github.com/izzahnin/disaster-risk-intelligence-backend/internal/model"
)

func TestCalculate_empty(t *testing.T) {
	result := Calculate([]model.Earthquake{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestCalculate_singleProvince(t *testing.T) {
	eq := []model.Earthquake{
		{Province: "Aceh", Magnitude: 5.5},
		{Province: "Aceh", Magnitude: 6.0},
	}
	result := Calculate(eq)
	if len(result) != 1 {
		t.Fatalf("expected 1 province, got %d", len(result))
	}
	if result[0].RiskScore != 100 {
		t.Errorf("single province risk_score = %v, want 100", result[0].RiskScore)
	}
	if result[0].Count != 2 {
		t.Errorf("count = %d, want 2", result[0].Count)
	}
}

func TestCalculate_sortDescending(t *testing.T) {
	// Aceh: count=10, avg=6.0 → skor lebih tinggi
	// Jawa: count=1, avg=4.5 → skor lebih rendah
	earthquakes := []model.Earthquake{}
	for i := 0; i < 10; i++ {
		earthquakes = append(earthquakes, model.Earthquake{Province: "Aceh", Magnitude: 6.0})
	}
	earthquakes = append(earthquakes, model.Earthquake{Province: "Jawa Timur", Magnitude: 4.5})

	result := Calculate(earthquakes)
	if len(result) != 2 {
		t.Fatalf("expected 2 provinces, got %d", len(result))
	}
	if result[0].Province != "Aceh" {
		t.Errorf("expected Aceh first (highest risk), got %s", result[0].Province)
	}
	if result[0].RiskScore <= result[1].RiskScore {
		t.Errorf("expected descending risk_score, got %v <= %v", result[0].RiskScore, result[1].RiskScore)
	}
}
