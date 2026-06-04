package appreleases

type capturedAudit struct {
	action     string
	targetType string
	targetID   string
	message    string
	metadata   map[string]any
}

func findCapturedAudit(audits []capturedAudit, action string) (capturedAudit, bool) {
	for _, audit := range audits {
		if audit.action == action {
			return audit, true
		}
	}
	return capturedAudit{}, false
}
