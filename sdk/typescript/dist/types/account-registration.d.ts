export declare const ACCOUNT_REGISTRATION_PROOF_DOMAIN: "zerone.auth/register-account/v1";
export type ZeroneAccountType = "agent" | "human" | "contract" | "system";
export interface AccountRegistrationProof {
    readonly chainId: string;
    readonly sender: string;
    readonly did: string;
    readonly identityPublicKey: Uint8Array;
    readonly accountType: ZeroneAccountType;
    readonly metadata: string;
}
/**
 * Returns the exact domain-separated bytes signed by the Ed25519 identity key
 * for MsgRegisterAccount. The independently signed Cosmos transaction does not
 * replace this proof of possession.
 *
 * The operational-key hash is deliberately absent: the chain derives that
 * SHA-256 commitment from identityPublicKey.
 */
export declare function accountRegistrationProofSignBytes(proof: AccountRegistrationProof): Uint8Array;
