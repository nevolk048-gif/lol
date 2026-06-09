package handlers

import "testing"

// Обработка webhook: события спора/чарджбэка распознаются, обычные платёжные — нет.
func TestIsDisputeEvent(t *testing.T) {
	disputeEvents := []string{
		"dispute.created",
		"dispute",
		"chargeback",
		"chargeback.created",
		"payment.chargeback",
		"payment.dispute",
	}
	for _, e := range disputeEvents {
		if !isDisputeEvent(e) {
			t.Errorf("expected %q to be recognized as a dispute event", e)
		}
	}

	nonDisputeEvents := []string{
		"payment.success",
		"payment.expired",
		"payout.success",
		"payout.error",
		"",
		"unknown",
	}
	for _, e := range nonDisputeEvents {
		if isDisputeEvent(e) {
			t.Errorf("expected %q NOT to be recognized as a dispute event", e)
		}
	}
}
