package social

import (
	"math"
	"reflect"
	"testing"
)

func pt(at string, v float64) GrowthPoint { return GrowthPoint{CapturedAt: at, Value: v} }

func near(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil || math.Abs(*got-want) > 1e-6 {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func isNil(t *testing.T, got *float64) {
	t.Helper()
	if got != nil {
		t.Fatalf("got %v, want nil", *got)
	}
}

func TestGrowthOver(t *testing.T) {
	isNil(t, GrowthOver(nil, 30, 0))
	isNil(t, GrowthOver([]GrowthPoint{pt("2026-06-12", 100)}, 30, 0))
	near(t, GrowthOver([]GrowthPoint{pt("2026-05-13", 200), pt("2026-05-27", 230), pt("2026-06-12", 250)}, 30, 0), 25)
	isNil(t, GrowthOver([]GrowthPoint{pt("2026-06-10", 100), pt("2026-06-12", 110)}, 30, 0))
	isNil(t, GrowthOver([]GrowthPoint{pt("2026-05-01", 0), pt("2026-06-12", 50)}, 30, 0))
	near(t, GrowthOver([]GrowthPoint{pt("2026-04-13", 1000), pt("2026-05-20", 1100), pt("2026-06-12", 1300)}, 60, 0), 30)
	isNil(t, GrowthOver([]GrowthPoint{pt("2026-04-14", 1000), pt("2026-06-12", 1300)}, 60, 0))
}

func TestGrowthAllTime(t *testing.T) {
	near(t, GrowthAllTime([]GrowthPoint{pt("2026-04-01", 100), pt("2026-05-01", 120), pt("2026-06-01", 150)}), 50)
	isNil(t, GrowthAllTime([]GrowthPoint{pt("2026-04-01", 100)}))
}

func TestWindowDeltaBaselineLag(t *testing.T) {
	s := []GrowthPoint{pt("2026-07-01", 10000), pt("2026-09-17", 24814)}
	if d := WindowDelta(s, 30, 7); d != nil {
		t.Fatalf("stale baseline accepted: %+v", d)
	}
	if d := WindowDelta(s, 30, 0); d == nil || d.Current != 24814 || d.Baseline != 10000 {
		t.Fatalf("unbounded = %+v", d)
	}
	if d := WindowDelta([]GrowthPoint{pt("2026-08-14", 20315), pt("2026-09-17", 24814)}, 30, 7); d == nil || d.Baseline != 20315 {
		t.Fatalf("within tolerance = %+v", d)
	}
	isNil(t, GrowthOver(s, 30, 7))
}

func TestMergeSeriesSum(t *testing.T) {
	got := MergeSeriesSum([][]GrowthPoint{{pt("2026-06-01", 100), pt("2026-06-10", 120)}, {pt("2026-06-05", 50)}})
	want := []GrowthPoint{pt("2026-06-01", 100), pt("2026-06-05", 150), pt("2026-06-10", 170)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	if len(MergeSeriesSum(nil)) != 0 || len(MergeSeriesSum([][]GrowthPoint{{}, {}})) != 0 {
		t.Fatal("empty")
	}
}
