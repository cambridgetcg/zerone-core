# R4-4 — Channels Module: Payment Channels

## Goal

Port x/channels — unidirectional payment channels for high-frequency
micro-payments between agents and tool providers. Supports off-chain state
updates, on-chain dispute resolution, and auto-settlement.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/channels/` — full module (4238 LOC keeper, 2976 LOC tests)
- `/Users/yuai/Desktop/legible_money/proto/legible/channels/v1/` — all protos

## Proto Files

### `proto/zerone/channels/v1/types.proto`
```protobuf
syntax = "proto3";
package zerone.channels.v1;
option go_package = "github.com/zerone-chain/zerone/x/channels/types";

message PaymentChannel {
  string channel_id            = 1;
  string payer                 = 2;   // bech32
  string receiver              = 3;   // bech32
  string deposited             = 4;   // uzrn total deposited
  string spent                 = 5;   // uzrn cumulative spent
  string available             = 6;   // deposited - spent
  string status                = 7;   // "open", "closing", "disputed", "settled"
  uint64 opened_at_block       = 8;
  uint64 expires_at_block      = 9;
  uint64 nonce                 = 10;
  string last_state_hash       = 11;
  uint64 settlement_frequency  = 12;  // blocks between auto-settlements
  uint64 last_settlement_block = 13;
  uint64 dispute_deadline      = 14;  // block by which dispute must resolve
  uint64 dispute_nonce         = 15;
}

message ChannelStateUpdate {
  string channel_id          = 1;
  uint64 nonce               = 2;
  string spent               = 3;    // cumulative uzrn
  string state_hash          = 4;
  string payer_signature     = 5;
  string receiver_signature  = 6;
}

message ChannelDispute {
  string channel_id              = 1;
  string disputer                = 2;
  uint64 disputed_nonce          = 3;
  string disputed_spent          = 4;
  string dispute_state_hash      = 5;
  uint64 deadline_block          = 6;
  bool   resolved                = 7;
  string resolution              = 8;  // "payer_wins", "receiver_wins", "split"
}
```

### `proto/zerone/channels/v1/tx.proto`
```protobuf
syntax = "proto3";
package zerone.channels.v1;
option go_package = "github.com/zerone-chain/zerone/x/channels/types";

import "cosmos/msg/v1/msg.proto";
import "zerone/channels/v1/types.proto";

service Msg {
  option (cosmos.msg.v1.service) = true;

  rpc OpenChannel(MsgOpenChannel) returns (MsgOpenChannelResponse);
  rpc DepositChannel(MsgDepositChannel) returns (MsgDepositChannelResponse);
  rpc UpdateState(MsgUpdateState) returns (MsgUpdateStateResponse);
  rpc CloseChannel(MsgCloseChannel) returns (MsgCloseChannelResponse);
  rpc DisputeChannel(MsgDisputeChannel) returns (MsgDisputeChannelResponse);
  rpc ClaimExpired(MsgClaimExpired) returns (MsgClaimExpiredResponse);
  rpc UpdateParams(MsgUpdateParams) returns (MsgUpdateParamsResponse);
}

message MsgOpenChannel {
  option (cosmos.msg.v1.signer) = "payer";
  string payer = 1; string receiver = 2;
  string deposit = 3; uint64 timeout_blocks = 4;
}
message MsgOpenChannelResponse { string channel_id = 1; }

message MsgDepositChannel {
  option (cosmos.msg.v1.signer) = "depositor";
  string depositor = 1; string channel_id = 2; string amount = 3;
}
message MsgDepositChannelResponse {}

message MsgUpdateState {
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1; string channel_id = 2;
  ChannelStateUpdate update = 3;
}
message MsgUpdateStateResponse {}

message MsgCloseChannel {
  option (cosmos.msg.v1.signer) = "closer";
  string closer = 1; string channel_id = 2;
  string final_spent = 3; uint64 final_nonce = 4;
  bytes counterparty_signature = 5;
}
message MsgCloseChannelResponse {}

message MsgDisputeChannel {
  option (cosmos.msg.v1.signer) = "disputer";
  string disputer = 1; string channel_id = 2;
  string claimed_spent = 3; uint64 claimed_nonce = 4;
  bytes proof_signature = 5;
}
message MsgDisputeChannelResponse {}

message MsgClaimExpired {
  option (cosmos.msg.v1.signer) = "claimer";
  string claimer = 1; string channel_id = 2;
}
message MsgClaimExpiredResponse { string refunded_amount = 1; }

message MsgUpdateParams {
  option (cosmos.msg.v1.signer) = "authority";
  string authority = 1; Params params = 2;
}
message MsgUpdateParamsResponse {}
```

### `proto/zerone/channels/v1/query.proto`
```protobuf
syntax = "proto3";
package zerone.channels.v1;
option go_package = "github.com/zerone-chain/zerone/x/channels/types";

import "google/api/annotations.proto";
import "zerone/channels/v1/types.proto";
import "zerone/channels/v1/genesis.proto";

service Query {
  rpc Channel(QueryChannelRequest) returns (QueryChannelResponse) {
    option (google.api.http).get = "/zerone/channels/v1/channel/{channel_id}";
  }
  rpc ChannelsByPayer(QueryByPayerRequest) returns (QueryByPayerResponse) {
    option (google.api.http).get = "/zerone/channels/v1/by_payer/{payer}";
  }
  rpc ChannelsByReceiver(QueryByReceiverRequest) returns (QueryByReceiverResponse) {
    option (google.api.http).get = "/zerone/channels/v1/by_receiver/{receiver}";
  }
  rpc Dispute(QueryDisputeRequest) returns (QueryDisputeResponse) {
    option (google.api.http).get = "/zerone/channels/v1/dispute/{channel_id}";
  }
  rpc Params(QueryParamsRequest) returns (QueryParamsResponse) {
    option (google.api.http).get = "/zerone/channels/v1/params";
  }
}

message QueryChannelRequest { string channel_id = 1; }
message QueryChannelResponse { PaymentChannel channel = 1; }
message QueryByPayerRequest { string payer = 1; }
message QueryByPayerResponse { repeated PaymentChannel channels = 1; }
message QueryByReceiverRequest { string receiver = 1; }
message QueryByReceiverResponse { repeated PaymentChannel channels = 1; }
message QueryDisputeRequest { string channel_id = 1; }
message QueryDisputeResponse { ChannelDispute dispute = 1; }
message QueryParamsRequest {}
message QueryParamsResponse { Params params = 1; }
```

### `proto/zerone/channels/v1/genesis.proto`
```protobuf
syntax = "proto3";
package zerone.channels.v1;
option go_package = "github.com/zerone-chain/zerone/x/channels/types";

import "zerone/channels/v1/types.proto";

message GenesisState {
  Params params = 1;
  repeated PaymentChannel channels = 2;
}

message Params {
  string min_deposit = 1;               // default: "1000000" (1 ZRN)
  uint64 min_timeout_blocks = 2;        // default: 100
  uint64 max_timeout_blocks = 3;        // default: 1000000
  uint64 dispute_window_blocks = 4;     // default: 500
  uint64 default_settlement_freq = 5;   // default: 100 blocks
  uint64 max_channels_per_pair = 6;     // default: 10
  string channel_open_fee = 7;          // default: "100000" (0.1 ZRN)
}
```

## Implementation

### Keeper

**State (`state.go`):**
- Channels: prefix `channel/` + channel_id
- Disputes: prefix `dispute/` + channel_id
- Payer index: prefix `ch-payer/` + payer + `/` + channel_id
- Receiver index: prefix `ch-receiver/` + receiver + `/` + channel_id
- Auto-increment channel ID counter

**Message Server:**

- `OpenChannel` — validate min deposit, check max_channels_per_pair, charge open fee, escrow deposit to module account, create channel with status "open"
- `DepositChannel` — payer only, add to deposited + available
- `UpdateState` — either party submits, verify signatures (both payer + receiver must sign the state), nonce must be > current nonce, spent must be ≤ deposited. This is the off-chain→on-chain state sync
- `CloseChannel` — cooperative close: requires counterparty signature on final state. Distribute: spent→receiver, remaining→payer. Mark settled
- `DisputeChannel` — either party. Verify signature proof. Start dispute window. Higher nonce wins. If disputer provides higher-nonce proof, they win
- `ClaimExpired` — payer claims funds from expired channel (block > expires_at_block). Refund available balance

**BeginBlocker:**
- Resolve disputes past deadline: if no counter-evidence submitted, disputer wins
- Auto-settle channels at settlement_frequency intervals: move spent→receiver on-chain

**Signature Verification:**
State updates require both parties to sign `hash(channel_id | nonce | spent | state_hash)`.
Use Ed25519 verification against the payer/receiver addresses.

### Tests
Port from draft (2976 LOC). Minimum:
- Open → deposit → update state → cooperative close
- Dispute resolution: higher nonce wins
- Dispute timeout: disputer wins by default
- Expired channel claim
- Signature verification (reject tampered updates)
- Auto-settlement in BeginBlocker
- Max channels per pair enforcement

## Conventions
- Token: uzrn. Module path: github.com/zerone-chain/zerone
- Signatures: Ed25519 over deterministic encoding of state
- Run `go build ./...` and `go test ./...` before finishing
