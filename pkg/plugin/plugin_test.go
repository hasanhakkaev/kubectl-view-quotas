package plugin

import (
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// TestBuildQuotaRows_AllZeroHard verifies that a ResourceQuota whose hard
// values are all 0 produces one row per resource instead of zero rows.
func TestBuildQuotaRows_AllZeroHard(t *testing.T) {
	quota := v1.ResourceQuota{}
	quota.Status.Hard = v1.ResourceList{
		"persistentvolumeclaims": resource.MustParse("0"),
		"services.loadbalancers": resource.MustParse("0"),
		"services.nodeports":     resource.MustParse("0"),
	}
	quota.Status.Used = v1.ResourceList{
		"persistentvolumeclaims": resource.MustParse("0"),
		"services.loadbalancers": resource.MustParse("0"),
		"services.nodeports":     resource.MustParse("0"),
	}

	rows := buildQuotaRows(quota)

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows for all-zero object quota, got %d", len(rows))
	}

	// Rows are sorted by resource name.
	expected := []string{"persistentvolumeclaims", "services.loadbalancers", "services.nodeports"}
	for i, row := range rows {
		if row.resource != expected[i] {
			t.Errorf("row %d: expected resource %q, got %q", i, expected[i], row.resource)
		}
		if row.used != "0" {
			t.Errorf("row %d (%s): expected used == \"0\", got %q", i, row.resource, row.used)
		}
		if row.hard != "0" {
			t.Errorf("row %d (%s): expected hard == \"0\", got %q", i, row.resource, row.hard)
		}
	}
}

// TestResourceUsage_ZeroUsedZeroHard verifies that used=0 and hard=0 renders
// as a full red bar at 100.0% (at-limit / blocked).
func TestResourceUsage_ZeroUsedZeroHard(t *testing.T) {
	color, usage := resourceUsage(0, 0)

	if color != colorRed {
		t.Errorf("resourceUsage(0,0): expected colorRed, got %v", color)
	}
	if !strings.Contains(usage, "100.0%") {
		t.Errorf("resourceUsage(0,0): expected usage to contain \"100.0%%\", got %q", usage)
	}
	// Full bar: all 10 block chars should be present.
	bar := strings.TrimRight(usage, " ")
	if !strings.HasPrefix(bar, strings.Repeat("█", 10)) {
		t.Errorf("resourceUsage(0,0): expected a full 10-char block bar, got %q", usage)
	}
	if strings.Contains(usage, "Inf") || strings.Contains(usage, "NaN") || strings.Contains(usage, "<none>") {
		t.Errorf("resourceUsage(0,0): output contains forbidden value: %q", usage)
	}
}

// TestResourceUsage_UsedOverZeroHard verifies that used>0 with hard=0 renders
// as over-limit (colorOver) with a full bar and no Inf/NaN.
func TestResourceUsage_UsedOverZeroHard(t *testing.T) {
	color, usage := resourceUsage(1, 0)

	if color != colorOver {
		t.Errorf("resourceUsage(1,0): expected colorOver, got %v", color)
	}
	if strings.Contains(usage, "Inf") || strings.Contains(usage, "NaN") {
		t.Errorf("resourceUsage(1,0): output contains forbidden value: %q", usage)
	}
	// Must contain the over-limit indicator.
	if !strings.Contains(usage, ">100%") {
		t.Errorf("resourceUsage(1,0): expected usage to contain \">100%%\", got %q", usage)
	}
}

// TestResourceUsage_Regression_FiveOfTen verifies that existing nonzero-hard
// behaviour is unchanged: 5 of 10 must still render as 50.0% green.
func TestResourceUsage_Regression_FiveOfTen(t *testing.T) {
	color, usage := resourceUsage(5, 10)

	if color != colorGreen {
		t.Errorf("resourceUsage(5,10): expected colorGreen, got %v", color)
	}
	if !strings.Contains(usage, "50.0%") {
		t.Errorf("resourceUsage(5,10): expected usage to contain \"50.0%%\", got %q", usage)
	}
}
