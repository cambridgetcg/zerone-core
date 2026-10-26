package cli

import (
	"bytes"
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// This is a read-only projection over existing ABCI keys, like the vindication
// queries. It adds no protobuf field, consensus key, or mutation endpoint.
type reformulationHoldResponse struct {
	SnapshotHeight     int64  `json:"snapshot_height"`
	Enabled            bool   `json:"enabled"`
	Found              bool   `json:"found"`
	AugmentationID     string `json:"augmentation_id"`
	BountyID           string `json:"bounty_id"`
	Held               bool   `json:"held"`
	Reason             string `json:"reason"`
	ProspectiveReason  string `json:"prospective_reason"`
	BountyEscrowLocked string `json:"bounty_escrow_locked"`
}

func queryReformulationHold(clientCtx client.Context, id string) (reformulationHoldResponse, error) {
	out := reformulationHoldResponse{AugmentationID: id, BountyEscrowLocked: "0"}
	marker, height, err := clientCtx.QueryStore([]byte(keeper.ReformulationVindicationOrderingEnabledStoreKey), types.StoreKey)
	if err != nil {
		return out, err
	}
	if height <= 0 || (clientCtx.Height != 0 && clientCtx.Height != height) {
		return out, fmt.Errorf("activation marker query returned an unexpected height %d", height)
	}
	out.SnapshotHeight = height
	if marker != nil && !bytes.Equal(marker, []byte{1}) {
		return out, fmt.Errorf("malformed reformulation and vindication ordering marker")
	}
	out.Enabled = bytes.Equal(marker, []byte{1})
	clientCtx = clientCtx.WithHeight(height)
	read := func(key []byte) ([]byte, error) {
		value, gotHeight, err := clientCtx.QueryStore(key, types.StoreKey)
		if err != nil {
			return nil, err
		}
		if gotHeight != height {
			return nil, fmt.Errorf("inconsistent query height: got %d, want %d", gotHeight, height)
		}
		return value, nil
	}
	bz, err := read(types.AugmentationKey(id))
	if err != nil {
		return out, err
	}
	if bz == nil {
		return out, nil
	}
	var aug types.Augmentation
	if err := proto.Unmarshal(bz, &aug); err != nil {
		return out, err
	}
	if aug.Id != id {
		return out, fmt.Errorf("augmentation key/id mismatch")
	}
	out.Found, out.BountyID = true, aug.BountyId
	bz, err = read(types.ParamsKey)
	if err != nil {
		return out, err
	}
	if len(bz) == 0 {
		return out, fmt.Errorf("knowledge params missing")
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return out, err
	}
	out.ProspectiveReason = keeper.ReformulationHoldReason(&aug, &params)
	if out.Enabled {
		out.Reason = out.ProspectiveReason
		out.Held = out.Reason != ""
	}
	if aug.BountyId != "" {
		bz, err = read(types.TrainingFundEscrowLockedKey(aug.BountyId))
		if err != nil {
			return out, err
		}
		if bz != nil {
			amount, ok := sdkmath.NewIntFromString(string(bz))
			if !ok || amount.IsNegative() {
				return out, fmt.Errorf("malformed bounty escrow")
			}
			out.BountyEscrowLocked = amount.String()
		}
	}
	return out, nil
}

func NewQueryReformulationHoldCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reformulation-hold [augmentation-id]",
		Short: "Query retained-ballot hold and pooled bounty escrow at one height",
		Long:  "Read the activation marker, retained ballots, params and escrow at one snapshot height. Before activation, prospective_reason supports a census but held is false. Bounty escrow is pooled, not an entitlement for this augmentation. A missing record or unreadable state is not a release authorization. Output is JSON; no new RPC or transaction is registered.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			result, err := queryReformulationHold(clientCtx, args[0])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
