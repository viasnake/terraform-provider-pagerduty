package util

import (
	"fmt"
	"strings"
)

// defaultMobilizationServiceErrSignals are the substrings the PagerDuty API is
// known to include in its error whenever an operation is rejected for targeting
// the account's Default Mobilization Service. Both the heimweh and PagerDuty
// go-pagerduty clients embed the API message in their Error() output, so a
// substring check identifies the condition regardless of which client produced
// the error, without an extra lookup call. The API uses inconsistent wording
// across endpoints, so we check for all known variants.
//
// Signals are matched case-insensitively against the full confirmed phrase
// rather than the bare words "triage service" — IsDefaultMobilizationServiceError
// also gates retry decisions (see the event_orchestration_path_* update paths),
// so a false positive there doesn't just show the wrong message, it can turn a
// genuinely retryable error into a hard failure. Matching only the exact
// observed phrases keeps this from tripping on an unrelated error that happens
// to mention a user-named service containing "triage service".
var defaultMobilizationServiceErrSignals = []string{
	"account default mobilization service",
	"cannot be created on a triage service",
	"cannot include the triage service",
}

// defaultMobilizationServiceSuffix is the shared trailing guidance appended to
// every Default Mobilization Service message so operators get consistent
// remediation advice across resources.
const defaultMobilizationServiceSuffix = "The Default Mobilization Service is a PagerDuty-managed service used for triage/investigations and cannot be modified or targeted by this operation. Remove this change from your configuration or target a different service."

// DefaultMobilizationServiceMsg is the operator-facing summary/detail pair for a
// single forbidden operation. Keeping summary and detail separate lets SDKv2
// resources render a single-string error while terraform-plugin-framework
// resources render the native summary/detail diagnostic.
type DefaultMobilizationServiceMsg struct {
	Summary string
	Detail  string
}

// Centralized wording for every affected operation. Adjusting operator-facing
// copy is a single-file edit here and never touches the detection or wiring.
var (
	DMSMsgServiceDelete = DefaultMobilizationServiceMsg{
		Summary: "Cannot delete the Default Mobilization Service",
		Detail:  "This service is managed by PagerDuty and cannot be deleted. If this resource was imported or created before the service became protected, remove it from Terraform state with `terraform state rm` instead of destroying it.",
	}

	DMSMsgServiceIntegrationCreate = DefaultMobilizationServiceMsg{
		Summary: "Cannot add an integration to the Default Mobilization Service",
		Detail:  "Integrations cannot be created on the Default Mobilization Service. Point this pagerduty_service_integration at a different service.",
	}

	DMSMsgOrchestrationRouter = DefaultMobilizationServiceMsg{
		Summary: "Cannot route events to the Default Mobilization Service",
		Detail:  "Event Orchestration router rules cannot target the Default Mobilization Service. Update the rule's route_to to a different service.",
	}

	DMSMsgOrchestrationService = DefaultMobilizationServiceMsg{
		Summary: "Cannot configure orchestration for the Default Mobilization Service",
		Detail:  "A service orchestration cannot be attached to the Default Mobilization Service.",
	}

	DMSMsgServiceDependency = DefaultMobilizationServiceMsg{
		Summary: "The Default Mobilization Service cannot be used in a service dependency",
		Detail:  "The Default Mobilization Service cannot be set as a supporting or dependent service. Remove it from this pagerduty_service_dependency.",
	}

	DMSMsgMaintenanceWindow = DefaultMobilizationServiceMsg{
		Summary: "Cannot schedule a maintenance window on the Default Mobilization Service",
		Detail:  "The Default Mobilization Service cannot be targeted by a maintenance window. Remove it from the services list.",
	}
)

// IsDefaultMobilizationServiceError reports whether err was returned by the API
// because the operation targeted the Default Mobilization Service.
func IsDefaultMobilizationServiceError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, signal := range defaultMobilizationServiceErrSignals {
		if strings.Contains(msg, signal) {
			return true
		}
	}
	return false
}

// Error renders the message as a single-string error for SDKv2 resources,
// preserving the original API error for debuggability.
func (m DefaultMobilizationServiceMsg) Error(cause error) error {
	return fmt.Errorf("%s. %s %s\n\nOriginal API error: %s", m.Summary, m.Detail, defaultMobilizationServiceSuffix, cause)
}

// Diagnostic renders the message as a (summary, detail) pair for
// terraform-plugin-framework resources, preserving the original API error in the
// detail for debuggability.
func (m DefaultMobilizationServiceMsg) Diagnostic(cause error) (summary, detail string) {
	detail = fmt.Sprintf("%s %s\n\nOriginal API error: %s", m.Detail, defaultMobilizationServiceSuffix, cause)
	return m.Summary, detail
}
