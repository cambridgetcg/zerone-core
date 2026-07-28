import { ChallengeOutcome } from "./types.js";
import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgSubmitChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgSubmitChallenge
 */
export interface MsgSubmitChallenge {
    challenger: string;
    domain: string;
    accusedValidators: string[];
    /**
     * uzrn
     */
    stake: string;
    reason: string;
}
/**
 * @name MsgSubmitChallengeResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgSubmitChallengeResponse
 */
export interface MsgSubmitChallengeResponse {
    challengeId: string;
}
/**
 * @name MsgAddEvidence
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgAddEvidence
 */
export interface MsgAddEvidence {
    challenger: string;
    challengeId: string;
    description: string;
    dataHash: string;
}
/**
 * @name MsgAddEvidenceResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgAddEvidenceResponse
 */
export interface MsgAddEvidenceResponse {
}
/**
 * @name MsgResolveChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgResolveChallenge
 */
export interface MsgResolveChallenge {
    authority: string;
    challengeId: string;
    outcome: ChallengeOutcome;
    reason: string;
}
/**
 * @name MsgResolveChallengeResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgResolveChallengeResponse
 */
export interface MsgResolveChallengeResponse {
}
/**
 * @name MsgFundBountyPool
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgFundBountyPool
 */
export interface MsgFundBountyPool {
    sender: string;
    domain: string;
    /**
     * uzrn
     */
    amount: string;
}
/**
 * @name MsgFundBountyPoolResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgFundBountyPoolResponse
 */
export interface MsgFundBountyPoolResponse {
}
/**
 * @name MsgUpdateParams
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgSubmitChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgSubmitChallenge
 */
export declare const MsgSubmitChallenge: {
    typeUrl: string;
    encode(message: MsgSubmitChallenge, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitChallenge;
    fromPartial(object: DeepPartial<MsgSubmitChallenge>): MsgSubmitChallenge;
};
/**
 * @name MsgSubmitChallengeResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgSubmitChallengeResponse
 */
export declare const MsgSubmitChallengeResponse: {
    typeUrl: string;
    encode(message: MsgSubmitChallengeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitChallengeResponse;
    fromPartial(object: DeepPartial<MsgSubmitChallengeResponse>): MsgSubmitChallengeResponse;
};
/**
 * @name MsgAddEvidence
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgAddEvidence
 */
export declare const MsgAddEvidence: {
    typeUrl: string;
    encode(message: MsgAddEvidence, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddEvidence;
    fromPartial(object: DeepPartial<MsgAddEvidence>): MsgAddEvidence;
};
/**
 * @name MsgAddEvidenceResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgAddEvidenceResponse
 */
export declare const MsgAddEvidenceResponse: {
    typeUrl: string;
    encode(_: MsgAddEvidenceResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddEvidenceResponse;
    fromPartial(_: DeepPartial<MsgAddEvidenceResponse>): MsgAddEvidenceResponse;
};
/**
 * @name MsgResolveChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgResolveChallenge
 */
export declare const MsgResolveChallenge: {
    typeUrl: string;
    encode(message: MsgResolveChallenge, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveChallenge;
    fromPartial(object: DeepPartial<MsgResolveChallenge>): MsgResolveChallenge;
};
/**
 * @name MsgResolveChallengeResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgResolveChallengeResponse
 */
export declare const MsgResolveChallengeResponse: {
    typeUrl: string;
    encode(_: MsgResolveChallengeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveChallengeResponse;
    fromPartial(_: DeepPartial<MsgResolveChallengeResponse>): MsgResolveChallengeResponse;
};
/**
 * @name MsgFundBountyPool
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgFundBountyPool
 */
export declare const MsgFundBountyPool: {
    typeUrl: string;
    encode(message: MsgFundBountyPool, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgFundBountyPool;
    fromPartial(object: DeepPartial<MsgFundBountyPool>): MsgFundBountyPool;
};
/**
 * @name MsgFundBountyPoolResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgFundBountyPoolResponse
 */
export declare const MsgFundBountyPoolResponse: {
    typeUrl: string;
    encode(_: MsgFundBountyPoolResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgFundBountyPoolResponse;
    fromPartial(_: DeepPartial<MsgFundBountyPoolResponse>): MsgFundBountyPoolResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
