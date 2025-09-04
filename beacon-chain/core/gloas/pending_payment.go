package gloas

import (
	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v6/config/params"
	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v6/runtime/version"
	"github.com/pkg/errors"
)

var (
	// errGloasNotActive is returned when Gloas fork functionality is used before fork activation.
	errGloasNotActive = errors.New("require Gloas fork activation")
)

// ProcessBuilderPendingPayments processes the builder pending payments from the previous epoch.
//
// Spec definition:
// def process_builder_pending_payments(state: BeaconState) -> None:
//
//	"""
//	Processes the builder pending payments from the previous epoch.
//	"""
//	quorum = get_builder_payment_quorum_threshold(state)
//	for payment in state.builder_pending_payments[:SLOTS_PER_EPOCH]:
//	    if payment.weight > quorum:
//	        exit_queue_epoch = compute_exit_epoch_and_update_churn(state, payment.withdrawal.amount)
//	        payment.withdrawal.withdrawable_epoch = Epoch(
//	            exit_queue_epoch + MIN_VALIDATOR_WITHDRAWABILITY_DELAY
//	        )
//	        state.builder_pending_withdrawals.append(payment.withdrawal)
//	state.builder_pending_payments = state.builder_pending_payments[SLOTS_PER_EPOCH:] + [
//	    BuilderPendingPayment() for _ in range(SLOTS_PER_EPOCH)
//	]
func ProcessBuilderPendingPayments(state state.BeaconState) error {
	if state.Version() < version.Gloas {
		return errGloasNotActive
	}

	quorum, err := getBuilderPaymentQuorumThreshold(state)
	if err != nil {
		return errors.Wrap(err, "could not compute builder payment quorum threshold")
	}

	payments, err := state.BuilderPendingPayments()
	if err != nil {
		return errors.Wrap(err, "could not get builder pending payments")
	}

	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	for i := uint64(0); i < slotsPerEpoch; i++ {
		payment := payments[i]
		if payment.Weight > quorum {
			exitQueueEpoch, err := state.ExitEpochAndUpdateChurn(payment.Withdrawal.Amount)
			if err != nil {
				return errors.Wrapf(err, "could not compute exit epoch for payment %d", i)
			}

			withdrawableEpoch, err := exitQueueEpoch.SafeAdd(uint64(params.BeaconConfig().MinValidatorWithdrawabilityDelay))
			if err != nil {
				return errors.Wrapf(err, "could not compute withdrawable epoch for payment %d", i)
			}
			payment.Withdrawal.WithdrawableEpoch = withdrawableEpoch

			if err := state.AppendBuilderPendingWithdrawal(payment.Withdrawal); err != nil {
				return errors.Wrapf(err, "could not append builder pending withdrawal %d", i)
			}
		}
	}

	if err := state.RotateBuilderPendingPayments(slotsPerEpoch); err != nil {
		return errors.Wrap(err, "could not rotate builder pending payments")
	}

	return nil
}

func getBuilderPaymentQuorumThreshold(state state.BeaconState) (primitives.Gwei, error) {
	totalActiveBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return 0, errors.Wrap(err, "could not get total active balance")
	}

	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	numerator := params.BeaconConfig().BuilderPaymentThresholdNumerator
	denominator := params.BeaconConfig().BuilderPaymentThresholdDenominator

	quorum := (totalActiveBalance / slotsPerEpoch) * numerator
	return primitives.Gwei(quorum / denominator), nil
}
