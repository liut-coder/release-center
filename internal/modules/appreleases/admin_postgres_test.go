package appreleases

import "testing"

func TestBuildReleaseQualityMetricsSplitsAPKAndResourceEvents(t *testing.T) {
	metrics := buildReleaseQualityMetrics(
		[]AppUpgradeEventAdmin{
			{EventType: "download_completed"},
			{EventType: "install_success"},
			{EventType: "install_failed", ErrorMessage: "installer denied"},
			{EventType: "checksum_failed"},
			{EventType: "download_started"},
		},
		[]AppUpgradeEventAdmin{
			{EventType: "activation_success"},
			{EventType: "activation_failed", ErrorMessage: "templates-bear checksum mismatch"},
			{EventType: "activation_failed", ErrorMessage: "templates-bear checksum mismatch"},
		},
		QualityPolicy{},
	)

	if len(metrics) != 2 {
		t.Fatalf("expected apk and resource metrics, got %+v", metrics)
	}

	apk := metrics[0]
	if apk.Category != "apk" || apk.TotalEvents != 5 || apk.SuccessEvents != 2 || apk.FailureEvents != 2 {
		t.Fatalf("unexpected apk metric: %+v", apk)
	}
	if apk.SuccessRate != 50 || apk.FailureRate != 50 {
		t.Fatalf("expected apk 50/50 rates, got %+v", apk)
	}
	if len(apk.FailureReasons) != 2 || apk.FailureReasons[0].Reason != "checksum_failed" || apk.FailureReasons[0].Count != 1 {
		t.Fatalf("unexpected apk failure reasons: %+v", apk.FailureReasons)
	}

	resource := metrics[1]
	if resource.Category != "resource" || resource.TotalEvents != 3 || resource.SuccessEvents != 1 || resource.FailureEvents != 2 {
		t.Fatalf("unexpected resource metric: %+v", resource)
	}
	if resource.SuccessRate != 33 || resource.FailureRate != 66 {
		t.Fatalf("expected integer resource rates, got %+v", resource)
	}
	if resource.LatestFailureReason != "templates-bear checksum mismatch" {
		t.Fatalf("unexpected latest failure reason: %+v", resource)
	}
	if len(resource.FailureReasons) != 1 || resource.FailureReasons[0].Count != 2 {
		t.Fatalf("unexpected resource failure reasons: %+v", resource.FailureReasons)
	}
	if resource.RecommendedAction != "observe" {
		t.Fatalf("expected resource metric to stay in observe below threshold, got %+v", resource)
	}
}

func TestBuildReleaseQualityMetricRecommendsActionsAtThresholds(t *testing.T) {
	resource := buildReleaseQualityMetric("resource", []AppUpgradeEventAdmin{
		{EventType: "activation_success"},
		{EventType: "activation_failed", ErrorMessage: "extract failed"},
		{EventType: "activation_failed", ErrorMessage: "extract failed"},
		{EventType: "activation_failed", ErrorMessage: "extract failed"},
	}, QualityPolicy{})
	if resource.RecommendedAction != "pause_resource" {
		t.Fatalf("expected resource pause recommendation, got %+v", resource)
	}

	apk := buildReleaseQualityMetric("apk", []AppUpgradeEventAdmin{
		{EventType: "download_completed"},
		{EventType: "checksum_failed"},
		{EventType: "checksum_failed"},
		{EventType: "checksum_failed"},
	}, QualityPolicy{})
	if apk.RecommendedAction != "rollback_apk" {
		t.Fatalf("expected apk rollback recommendation, got %+v", apk)
	}

	alerts := buildQualityAlerts([]ReleaseQualityMetric{resource, apk})
	if len(alerts) != 2 {
		t.Fatalf("expected two quality alerts, got %+v", alerts)
	}
	if alerts[0].RecommendedAction != "pause_resource" || alerts[0].Severity != "warning" {
		t.Fatalf("unexpected resource alert: %+v", alerts[0])
	}
	if alerts[1].RecommendedAction != "rollback_apk" || alerts[1].Severity != "critical" {
		t.Fatalf("unexpected apk alert: %+v", alerts[1])
	}
}

func TestBuildReleaseQualityMetricUsesConfigurablePolicy(t *testing.T) {
	policy := QualityPolicy{
		ResourceActivationFailedCount: 4,
		ResourceFailureRate:           80,
		APKChecksumFailedCount:        5,
		APKInstallFailedCount:         10,
		APKInstallFailureRate:         10,
	}
	resource := buildReleaseQualityMetric("resource", []AppUpgradeEventAdmin{
		{EventType: "activation_success"},
		{EventType: "activation_failed"},
		{EventType: "activation_failed"},
		{EventType: "activation_failed"},
	}, policy)
	if resource.RecommendedAction != "observe" {
		t.Fatalf("expected custom resource policy to keep observing, got %+v", resource)
	}
	if resource.PolicyThreshold != "activation_failed >= 4 且失败率 >= 80%" {
		t.Fatalf("unexpected resource threshold: %+v", resource)
	}

	apk := buildReleaseQualityMetric("apk", []AppUpgradeEventAdmin{
		{EventType: "checksum_failed"},
		{EventType: "checksum_failed"},
		{EventType: "checksum_failed"},
	}, policy)
	if apk.RecommendedAction != "observe" {
		t.Fatalf("expected custom apk policy to keep observing, got %+v", apk)
	}
	if alerts := buildQualityAlerts([]ReleaseQualityMetric{resource, apk}); len(alerts) != 0 {
		t.Fatalf("expected no alerts below custom thresholds, got %+v", alerts)
	}
}
