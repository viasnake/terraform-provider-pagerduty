package util

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestIsDefaultMobilizationServiceError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "unrelated error",
			err:  errors.New("DELETE API call to https://api.pagerduty.com/services/PXXXXXX failed 404 Not Found. Code: 0, Errors: [], Message: Not Found"),
			want: false,
		},
		{
			// Mirrors the heimweh client's Error() output used by the legacy provider.
			name: "heimweh-style forbidden error",
			err:  errors.New("DELETE API call to https://api.pagerduty.com/services/PXXXXXX failed 403 Forbidden. Code: 0, Errors: [Account Default Mobilization Service cannot be deleted], Message: Forbidden"),
			want: true,
		},
		{
			// Mirrors the PagerDuty/go-pagerduty client's Error() output used by the plugin provider.
			name: "go-pagerduty-style bad request error",
			err:  errors.New("HTTP response failed with status code 400, message: Bad Request (code: 2001): Account Default Mobilization Service is not a valid target"),
			want: true,
		},
		{
			name: "wrapped forbidden error",
			err:  fmt.Errorf("Error reading: PXXXXXX: %w", errors.New("failed 403 Forbidden. Errors: [Account Default Mobilization Service cannot be deleted]")),
			want: true,
		},
		{
			// Confirmed against a live 422 from POST .../services/PXXXXXX/integrations:
			// "Errors: [Integrations cannot be created on a triage service]".
			name: "triage service wording - integration create",
			err:  errors.New("POST API call to https://api.pd-staging.com/services/PXXXXXX/integrations failed 422 Unprocessable Entity. Code: 2001, Errors: [Integrations cannot be created on a triage service], Message: Invalid Input Provided"),
			want: true,
		},
		{
			// Confirmed against a live 422 from POST .../maintenance_windows:
			// "Errors: [Maintenance windows cannot include the triage service.]".
			name: "triage service wording - maintenance window create",
			err:  errors.New("POST API call to https://api.pd-staging.com/maintenance_windows failed 422 Unprocessable Entity. Code: 2001, Errors: [Maintenance windows cannot include the triage service.], Message: Invalid Input Provided"),
			want: true,
		},
		{
			// A user-named service can legitimately contain the bare words
			// "triage service" without this being a Default Mobilization Service
			// rejection. Since IsDefaultMobilizationServiceError also gates retry
			// decisions in the event_orchestration_path_* update paths, matching
			// only the confirmed full phrases (not the bare words) keeps this from
			// misclassifying an unrelated, genuinely retryable error.
			name: "unrelated error mentioning a user-named triage service",
			err:  errors.New("PUT API call to https://api.pagerduty.com/services/PXXXXXX failed 422 Unprocessable Entity. Code: 2001, Errors: [Escalation policy is invalid for payments triage service], Message: Invalid Input Provided"),
			want: false,
		},
		{
			// Detection must be case-insensitive, since API wording casing isn't guaranteed.
			name: "triage service wording - different casing",
			err:  errors.New("Errors: [Integrations cannot be created on a Triage Service]"),
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsDefaultMobilizationServiceError(tc.err); got != tc.want {
				t.Errorf("IsDefaultMobilizationServiceError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestDefaultMobilizationServiceMsgError(t *testing.T) {
	cause := errors.New("failed 403 Forbidden. Errors: [Account Default Mobilization Service cannot be deleted]")
	got := DMSMsgServiceDelete.Error(cause).Error()

	for _, want := range []string{
		DMSMsgServiceDelete.Summary,
		DMSMsgServiceDelete.Detail,
		defaultMobilizationServiceSuffix,
		"Original API error: " + cause.Error(),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Error() = %q, expected it to contain %q", got, want)
		}
	}
}

func TestDefaultMobilizationServiceMsgDiagnostic(t *testing.T) {
	cause := errors.New("HTTP response failed with status code 400: Account Default Mobilization Service is not a valid target")
	summary, detail := DMSMsgServiceDependency.Diagnostic(cause)

	if summary != DMSMsgServiceDependency.Summary {
		t.Errorf("summary = %q, want %q", summary, DMSMsgServiceDependency.Summary)
	}

	for _, want := range []string{
		DMSMsgServiceDependency.Detail,
		defaultMobilizationServiceSuffix,
		"Original API error: " + cause.Error(),
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail = %q, expected it to contain %q", detail, want)
		}
	}
}
