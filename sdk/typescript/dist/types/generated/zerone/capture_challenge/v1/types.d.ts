import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/** ChallengeStatus represents the lifecycle of a capture challenge. */
export declare enum ChallengeStatus {
    CHALLENGE_STATUS_UNSPECIFIED = 0,
    CHALLENGE_STATUS_OPEN = 1,
    CHALLENGE_STATUS_EVIDENCE = 2,
    CHALLENGE_STATUS_UNDER_REVIEW = 3,
    CHALLENGE_STATUS_RESOLVED = 4,
    CHALLENGE_STATUS_EXPIRED = 5,
    UNRECOGNIZED = -1
}
export declare function challengeStatusFromJSON(object: any): ChallengeStatus;
export declare function challengeStatusToJSON(object: ChallengeStatus): string;
/** ChallengeOutcome represents the resolution of a challenge. */
export declare enum ChallengeOutcome {
    CHALLENGE_OUTCOME_UNSPECIFIED = 0,
    CHALLENGE_OUTCOME_UPHELD = 1,
    CHALLENGE_OUTCOME_REJECTED = 2,
    CHALLENGE_OUTCOME_PARTIAL = 3,
    UNRECOGNIZED = -1
}
export declare function challengeOutcomeFromJSON(object: any): ChallengeOutcome;
export declare function challengeOutcomeToJSON(object: ChallengeOutcome): string;
/**
 * CaptureEvidence is a piece of evidence attached to a challenge.
 * @name CaptureEvidence
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureEvidence
 */
export interface CaptureEvidence {
    description: string;
    dataHash: string;
    submittedBlock: bigint;
}
/**
 * CaptureResolution records how a challenge was resolved.
 * @name CaptureResolution
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureResolution
 */
export interface CaptureResolution {
    outcome: ChallengeOutcome;
    /**
     * authority address
     */
    resolver: string;
    reason: string;
    resolvedBlock: bigint;
    /**
     * uzrn
     */
    rewardAmount: string;
    /**
     * uzrn
     */
    slashAmount: string;
}
/**
 * ValidatorSlash records a slash applied to an accused validator.
 * @name ValidatorSlash
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.ValidatorSlash
 */
export interface ValidatorSlash {
    validator: string;
    /**
     * uzrn
     */
    slashAmount: string;
    reason: string;
}
/**
 * CaptureChallenge is the main challenge record.
 * @name CaptureChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureChallenge
 */
export interface CaptureChallenge {
    id: string;
    challenger: string;
    domain: string;
    accusedValidators: string[];
    /**
     * uzrn
     */
    stake: string;
    status: ChallengeStatus;
    evidence: CaptureEvidence[];
    createdBlock: bigint;
    evidenceDeadline: bigint;
    reviewDeadline: bigint;
    resolution?: CaptureResolution;
    slashes: ValidatorSlash[];
}
/**
 * DomainBountyPool holds the bounty fund for a domain.
 * @name DomainBountyPool
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.DomainBountyPool
 */
export interface DomainBountyPool {
    domain: string;
    /**
     * uzrn
     */
    balance: string;
}
/**
 * CaptureEvidence is a piece of evidence attached to a challenge.
 * @name CaptureEvidence
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureEvidence
 */
export declare const CaptureEvidence: {
    typeUrl: string;
    encode(message: CaptureEvidence, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CaptureEvidence;
    fromPartial(object: DeepPartial<CaptureEvidence>): CaptureEvidence;
};
/**
 * CaptureResolution records how a challenge was resolved.
 * @name CaptureResolution
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureResolution
 */
export declare const CaptureResolution: {
    typeUrl: string;
    encode(message: CaptureResolution, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CaptureResolution;
    fromPartial(object: DeepPartial<CaptureResolution>): CaptureResolution;
};
/**
 * ValidatorSlash records a slash applied to an accused validator.
 * @name ValidatorSlash
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.ValidatorSlash
 */
export declare const ValidatorSlash: {
    typeUrl: string;
    encode(message: ValidatorSlash, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ValidatorSlash;
    fromPartial(object: DeepPartial<ValidatorSlash>): ValidatorSlash;
};
/**
 * CaptureChallenge is the main challenge record.
 * @name CaptureChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureChallenge
 */
export declare const CaptureChallenge: {
    typeUrl: string;
    encode(message: CaptureChallenge, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CaptureChallenge;
    fromPartial(object: DeepPartial<CaptureChallenge>): CaptureChallenge;
};
/**
 * DomainBountyPool holds the bounty fund for a domain.
 * @name DomainBountyPool
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.DomainBountyPool
 */
export declare const DomainBountyPool: {
    typeUrl: string;
    encode(message: DomainBountyPool, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): DomainBountyPool;
    fromPartial(object: DeepPartial<DomainBountyPool>): DomainBountyPool;
};
