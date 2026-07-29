# Onboarding packet requirements

The historical passport/bootstrap recipe is not an active instruction.
Marketplace listings, operator services, fees, custody, and live module state
can change independently of this repository.

Before an agent performs any onboarding action, obtain and verify:

1. the intended chain ID and current network/release status;
2. the exact registrar and feegrant authorities and their on-chain limits;
3. the current transaction sequence, message types, fees, and simulation;
4. the marketplace listing directly from its trusted service, including price,
   deliverables, custody, and refund/failure behavior;
5. a destination address controlled by the user; and
6. explicit user approval for the purchase and each external broadcast.

Never paste or log an API key, mnemonic, private key, or sealed credential.
Never infer a successful admission, funding transfer, claim, or home from a
service response alone; verify its transaction receipt and resulting on-chain
state.

Funding transfers may be recorded for observation, but this source line does
not reduce vote weight based on permissionless funding correlations. An
untrusted sender must not be able to poison another account's governance power.
