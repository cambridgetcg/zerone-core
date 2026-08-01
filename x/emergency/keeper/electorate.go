package keeper

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

const ceremonyElectorateSnapshotVersion = types.ElectorateSnapshotVersionV1

// buildCeremonyElectorateSnapshot freezes the exact custom Guardian/council
// electorate and decision policy for one ceremony. In-flight authority must
// not change when staking state, bootstrap-council expiry, or params change.
func (k Keeper) buildCeremonyElectorateSnapshot(
	ctx context.Context,
	params *types.Params,
	ceremonyType types.CeremonyType,
) ([]*types.EmergencyElectorateMember, *big.Int, uint64, error) {
	guardians, err := k.stakingKeeper.GetGuardianValidators(ctx)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("read guardian electorate: %w", err)
	}

	powers := make(map[string]*big.Int)
	for _, guardian := range guardians {
		if !guardian.IsActive || guardian.Tier != types.TierGuardian {
			continue
		}
		if strings.TrimSpace(guardian.Address) != guardian.Address || guardian.Address == "" {
			return nil, nil, 0, fmt.Errorf("guardian electorate contains an invalid address")
		}
		if _, err := sdk.AccAddressFromBech32(guardian.Address); err != nil {
			return nil, nil, 0, fmt.Errorf(
				"guardian electorate contains unsignable account address %q: %w",
				guardian.Address,
				err,
			)
		}
		power, ok := new(big.Int).SetString(guardian.TotalStake, 10)
		if !ok || power.Sign() <= 0 {
			return nil, nil, 0, fmt.Errorf(
				"guardian %q has invalid effective stake %q",
				guardian.Address,
				guardian.TotalStake,
			)
		}
		if _, duplicate := powers[guardian.Address]; duplicate {
			return nil, nil, 0, fmt.Errorf("guardian electorate contains duplicate address %q", guardian.Address)
		}
		powers[guardian.Address] = power
	}

	councilActive := k.isCouncilActive(ctx, params)
	if councilActive {
		virtualPower, ok := new(big.Int).SetString(params.CouncilVirtualStake, 10)
		if !ok || virtualPower.Sign() <= 0 {
			return nil, nil, 0, fmt.Errorf("genesis council has invalid virtual stake %q", params.CouncilVirtualStake)
		}
		for _, member := range params.GenesisCouncil {
			if _, err := sdk.AccAddressFromBech32(member); err != nil {
				return nil, nil, 0, fmt.Errorf(
					"genesis council electorate contains unsignable account address %q: %w",
					member,
					err,
				)
			}
			if _, guardian := powers[member]; guardian {
				continue
			}
			powers[member] = new(big.Int).Set(virtualPower)
		}
	}
	if len(powers) > types.MaxEmergencyElectorateSize {
		return nil, nil, 0, fmt.Errorf(
			"emergency electorate has %d eligible addresses, exceeds consensus maximum %d",
			len(powers),
			types.MaxEmergencyElectorateSize,
		)
	}

	addresses := make([]string, 0, len(powers))
	for address := range powers {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)

	total := new(big.Int)
	electorate := make([]*types.EmergencyElectorateMember, 0, len(addresses))
	for _, address := range addresses {
		power := powers[address]
		total.Add(total, power)
		electorate = append(electorate, &types.EmergencyElectorateMember{
			Address: address,
			Power:   power.String(),
		})
	}
	if total.Sign() <= 0 {
		return nil, nil, 0, fmt.Errorf("emergency electorate has no positive voting power")
	}
	if !councilActive {
		minPower, ok := new(big.Int).SetString(params.MinGuardianStake, 10)
		if !ok || minPower.Sign() <= 0 {
			return nil, nil, 0, fmt.Errorf("invalid min_guardian_stake %q", params.MinGuardianStake)
		}
		if total.Cmp(minPower) < 0 {
			return nil, nil, 0, fmt.Errorf(
				"total guardian stake %s is below required minimum %s",
				total,
				minPower,
			)
		}
	}

	threshold := getQuorumThreshold(ceremonyType, params)
	if threshold == 0 || threshold > 1_000_000 {
		return nil, nil, 0, fmt.Errorf("invalid %s quorum threshold %d", ceremonyType, threshold)
	}
	if params.MinDistinctVoters == 0 {
		return nil, nil, 0, fmt.Errorf("min_distinct_voters must be positive")
	}
	if uint64(len(electorate)) < params.MinDistinctVoters {
		return nil, nil, 0, fmt.Errorf(
			"electorate has %d eligible addresses, below min_distinct_voters %d",
			len(electorate),
			params.MinDistinctVoters,
		)
	}
	return electorate, total, threshold, nil
}

// validateCeremonySnapshot verifies the immutable electorate and returns an
// address-to-power index. It intentionally rejects incomplete legacy
// snapshots; callers terminalize such non-terminal ceremonies without
// applying their effects.
func validateCeremonySnapshot(
	ceremony *types.EmergencyCeremony,
) (map[string]*big.Int, *big.Int, error) {
	if ceremony.ElectorateSnapshotVersion != ceremonyElectorateSnapshotVersion {
		return nil, nil, fmt.Errorf(
			"unsupported electorate snapshot version %d",
			ceremony.ElectorateSnapshotVersion,
		)
	}
	if ceremony.QuorumThreshold == 0 || ceremony.QuorumThreshold > 1_000_000 {
		return nil, nil, fmt.Errorf("invalid snapshotted quorum threshold %d", ceremony.QuorumThreshold)
	}
	if ceremony.MinDistinctVoters == 0 {
		return nil, nil, fmt.Errorf("invalid snapshotted min distinct voters")
	}
	if len(ceremony.Electorate) == 0 {
		return nil, nil, fmt.Errorf("empty electorate snapshot")
	}
	if len(ceremony.Electorate) > types.MaxEmergencyElectorateSize {
		return nil, nil, fmt.Errorf(
			"electorate has %d members, exceeds consensus maximum %d",
			len(ceremony.Electorate),
			types.MaxEmergencyElectorateSize,
		)
	}
	if uint64(len(ceremony.Electorate)) < ceremony.MinDistinctVoters {
		return nil, nil, fmt.Errorf(
			"electorate has %d members, below snapshotted min distinct voters %d",
			len(ceremony.Electorate),
			ceremony.MinDistinctVoters,
		)
	}

	expectedTotal, ok := new(big.Int).SetString(ceremony.ElectorateTotalPower, 10)
	if !ok || expectedTotal.Sign() <= 0 {
		return nil, nil, fmt.Errorf("invalid electorate total power %q", ceremony.ElectorateTotalPower)
	}

	index := make(map[string]*big.Int, len(ceremony.Electorate))
	actualTotal := new(big.Int)
	previousAddress := ""
	for i, member := range ceremony.Electorate {
		if member == nil {
			return nil, nil, fmt.Errorf("nil electorate member at index %d", i)
		}
		if member.Address == "" ||
			strings.TrimSpace(member.Address) != member.Address ||
			(i > 0 && member.Address <= previousAddress) {
			return nil, nil, fmt.Errorf("electorate must contain unique addresses in strict lexical order")
		}
		if _, err := sdk.AccAddressFromBech32(member.Address); err != nil {
			return nil, nil, fmt.Errorf(
				"electorate member %q is not a signable account address: %w",
				member.Address,
				err,
			)
		}
		power, ok := new(big.Int).SetString(member.Power, 10)
		if !ok || power.Sign() <= 0 {
			return nil, nil, fmt.Errorf("electorate member %q has invalid power %q", member.Address, member.Power)
		}
		index[member.Address] = power
		actualTotal.Add(actualTotal, power)
		previousAddress = member.Address
	}
	if actualTotal.Cmp(expectedTotal) != 0 {
		return nil, nil, fmt.Errorf(
			"electorate total power mismatch: members=%s stored=%s",
			actualTotal,
			expectedTotal,
		)
	}
	return index, expectedTotal, nil
}

// validateCeremonyTallies proves that persisted aggregate fields are exactly
// the sum of unique votes from the immutable electorate. This prevents a
// forged or corrupted genesis/export record from manufacturing quorum.
func validateCeremonyTallies(
	ceremony *types.EmergencyCeremony,
	electorate map[string]*big.Int,
) error {
	yes := new(big.Int)
	no := new(big.Int)
	seenPrevotes := make(map[string]struct{}, len(ceremony.Prevotes))
	approved := make(map[string]struct{}, len(ceremony.Prevotes))
	for i, entry := range ceremony.Prevotes {
		if entry == nil || entry.Value == nil {
			return fmt.Errorf("nil prevote at index %d", i)
		}
		if entry.Key == "" || entry.Key != entry.Value.Voter {
			return fmt.Errorf("prevote key/voter mismatch at index %d", i)
		}
		if _, duplicate := seenPrevotes[entry.Key]; duplicate {
			return fmt.Errorf("duplicate prevote from %q", entry.Key)
		}
		power, eligible := electorate[entry.Key]
		if !eligible {
			return fmt.Errorf("prevote from address %q outside electorate snapshot", entry.Key)
		}
		seenPrevotes[entry.Key] = struct{}{}
		if entry.Value.Approve {
			yes.Add(yes, power)
			approved[entry.Key] = struct{}{}
		} else {
			no.Add(no, power)
		}
	}

	precommit := new(big.Int)
	seenPrecommits := make(map[string]struct{}, len(ceremony.Precommits))
	for i, entry := range ceremony.Precommits {
		if entry == nil || entry.Value == nil {
			return fmt.Errorf("nil precommit at index %d", i)
		}
		if entry.Key == "" || entry.Key != entry.Value.Voter {
			return fmt.Errorf("precommit key/voter mismatch at index %d", i)
		}
		if _, duplicate := seenPrecommits[entry.Key]; duplicate {
			return fmt.Errorf("duplicate precommit from %q", entry.Key)
		}
		if _, yesPrevote := approved[entry.Key]; !yesPrevote {
			return fmt.Errorf("precommit from %q lacks an approving prevote", entry.Key)
		}
		power, eligible := electorate[entry.Key]
		if !eligible {
			return fmt.Errorf("precommit from address %q outside electorate snapshot", entry.Key)
		}
		seenPrecommits[entry.Key] = struct{}{}
		precommit.Add(precommit, power)
	}

	stored := []struct {
		name  string
		value string
		want  *big.Int
	}{
		{"yes prevote stake", ceremony.YesPrevoteStake, yes},
		{"no prevote stake", ceremony.NoPrevoteStake, no},
		{"precommit stake", ceremony.PrecommitStake, precommit},
	}
	for _, tally := range stored {
		got, ok := new(big.Int).SetString(tally.value, 10)
		if !ok || got.Sign() < 0 {
			return fmt.Errorf("invalid %s %q", tally.name, tally.value)
		}
		if got.Cmp(tally.want) != 0 {
			return fmt.Errorf("%s mismatch: votes=%s stored=%s", tally.name, tally.want, got)
		}
	}
	return nil
}
