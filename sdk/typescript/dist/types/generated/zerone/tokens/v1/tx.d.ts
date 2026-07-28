import { TokenFeatures } from "./types.js";
import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgCreateToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateToken
 */
export interface MsgCreateToken {
    creator: string;
    name: string;
    symbol: string;
    decimals: number;
    initialSupply: string;
    /**
     * "0" = unlimited
     */
    maxSupply: string;
    features?: TokenFeatures;
}
/**
 * @name MsgCreateTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateTokenResponse
 */
export interface MsgCreateTokenResponse {
    tokenId: string;
}
/**
 * @name MsgMintToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgMintToken
 */
export interface MsgMintToken {
    authority: string;
    tokenId: string;
    to: string;
    amount: string;
}
/**
 * @name MsgMintTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgMintTokenResponse
 */
export interface MsgMintTokenResponse {
}
/**
 * @name MsgBurnToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgBurnToken
 */
export interface MsgBurnToken {
    burner: string;
    tokenId: string;
    amount: string;
}
/**
 * @name MsgBurnTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgBurnTokenResponse
 */
export interface MsgBurnTokenResponse {
}
/**
 * @name MsgTransferToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferToken
 */
export interface MsgTransferToken {
    sender: string;
    tokenId: string;
    to: string;
    amount: string;
}
/**
 * @name MsgTransferTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferTokenResponse
 */
export interface MsgTransferTokenResponse {
}
/**
 * @name MsgApproveToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgApproveToken
 */
export interface MsgApproveToken {
    owner: string;
    tokenId: string;
    spender: string;
    amount: string;
}
/**
 * @name MsgApproveTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgApproveTokenResponse
 */
export interface MsgApproveTokenResponse {
}
/**
 * @name MsgTransferFrom
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferFrom
 */
export interface MsgTransferFrom {
    spender: string;
    tokenId: string;
    from: string;
    to: string;
    amount: string;
}
/**
 * @name MsgTransferFromResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferFromResponse
 */
export interface MsgTransferFromResponse {
}
/**
 * @name MsgPauseToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgPauseToken
 */
export interface MsgPauseToken {
    authority: string;
    tokenId: string;
}
/**
 * @name MsgPauseTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgPauseTokenResponse
 */
export interface MsgPauseTokenResponse {
}
/**
 * @name MsgUnpauseToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnpauseToken
 */
export interface MsgUnpauseToken {
    authority: string;
    tokenId: string;
}
/**
 * @name MsgUnpauseTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnpauseTokenResponse
 */
export interface MsgUnpauseTokenResponse {
}
/**
 * @name MsgDelegatePower
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgDelegatePower
 */
export interface MsgDelegatePower {
    delegator: string;
    tokenId: string;
    delegate: string;
    amount: string;
}
/**
 * @name MsgDelegatePowerResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgDelegatePowerResponse
 */
export interface MsgDelegatePowerResponse {
}
/**
 * @name MsgUndelegatePower
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUndelegatePower
 */
export interface MsgUndelegatePower {
    delegator: string;
    tokenId: string;
    delegate: string;
    amount: string;
}
/**
 * @name MsgUndelegatePowerResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUndelegatePowerResponse
 */
export interface MsgUndelegatePowerResponse {
}
/**
 * @name MsgWrapToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgWrapToken
 */
export interface MsgWrapToken {
    sender: string;
    tokenId: string;
    amount: string;
}
/**
 * @name MsgWrapTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgWrapTokenResponse
 */
export interface MsgWrapTokenResponse {
    wrappedDenom: string;
}
/**
 * @name MsgUnwrapToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnwrapToken
 */
export interface MsgUnwrapToken {
    sender: string;
    wrappedDenom: string;
    amount: string;
}
/**
 * @name MsgUnwrapTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnwrapTokenResponse
 */
export interface MsgUnwrapTokenResponse {
    tokenId: string;
}
/**
 * @name MsgCreateEmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateEmissionPeriod
 */
export interface MsgCreateEmissionPeriod {
    authority: string;
    startBlock: bigint;
    endBlock: bigint;
    /**
     * uzrn per block
     */
    amountPerBlock: string;
    /**
     * module account or address
     */
    recipient: string;
}
/**
 * @name MsgCreateEmissionPeriodResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateEmissionPeriodResponse
 */
export interface MsgCreateEmissionPeriodResponse {
    emissionId: string;
}
/**
 * @name MsgCancelEmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCancelEmissionPeriod
 */
export interface MsgCancelEmissionPeriod {
    authority: string;
    emissionId: string;
}
/**
 * @name MsgCancelEmissionPeriodResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCancelEmissionPeriodResponse
 */
export interface MsgCancelEmissionPeriodResponse {
}
/**
 * @name MsgUpdateParams
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgCreateToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateToken
 */
export declare const MsgCreateToken: {
    typeUrl: string;
    encode(message: MsgCreateToken, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateToken;
    fromPartial(object: DeepPartial<MsgCreateToken>): MsgCreateToken;
};
/**
 * @name MsgCreateTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateTokenResponse
 */
export declare const MsgCreateTokenResponse: {
    typeUrl: string;
    encode(message: MsgCreateTokenResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateTokenResponse;
    fromPartial(object: DeepPartial<MsgCreateTokenResponse>): MsgCreateTokenResponse;
};
/**
 * @name MsgMintToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgMintToken
 */
export declare const MsgMintToken: {
    typeUrl: string;
    encode(message: MsgMintToken, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgMintToken;
    fromPartial(object: DeepPartial<MsgMintToken>): MsgMintToken;
};
/**
 * @name MsgMintTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgMintTokenResponse
 */
export declare const MsgMintTokenResponse: {
    typeUrl: string;
    encode(_: MsgMintTokenResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgMintTokenResponse;
    fromPartial(_: DeepPartial<MsgMintTokenResponse>): MsgMintTokenResponse;
};
/**
 * @name MsgBurnToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgBurnToken
 */
export declare const MsgBurnToken: {
    typeUrl: string;
    encode(message: MsgBurnToken, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgBurnToken;
    fromPartial(object: DeepPartial<MsgBurnToken>): MsgBurnToken;
};
/**
 * @name MsgBurnTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgBurnTokenResponse
 */
export declare const MsgBurnTokenResponse: {
    typeUrl: string;
    encode(_: MsgBurnTokenResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgBurnTokenResponse;
    fromPartial(_: DeepPartial<MsgBurnTokenResponse>): MsgBurnTokenResponse;
};
/**
 * @name MsgTransferToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferToken
 */
export declare const MsgTransferToken: {
    typeUrl: string;
    encode(message: MsgTransferToken, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgTransferToken;
    fromPartial(object: DeepPartial<MsgTransferToken>): MsgTransferToken;
};
/**
 * @name MsgTransferTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferTokenResponse
 */
export declare const MsgTransferTokenResponse: {
    typeUrl: string;
    encode(_: MsgTransferTokenResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgTransferTokenResponse;
    fromPartial(_: DeepPartial<MsgTransferTokenResponse>): MsgTransferTokenResponse;
};
/**
 * @name MsgApproveToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgApproveToken
 */
export declare const MsgApproveToken: {
    typeUrl: string;
    encode(message: MsgApproveToken, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgApproveToken;
    fromPartial(object: DeepPartial<MsgApproveToken>): MsgApproveToken;
};
/**
 * @name MsgApproveTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgApproveTokenResponse
 */
export declare const MsgApproveTokenResponse: {
    typeUrl: string;
    encode(_: MsgApproveTokenResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgApproveTokenResponse;
    fromPartial(_: DeepPartial<MsgApproveTokenResponse>): MsgApproveTokenResponse;
};
/**
 * @name MsgTransferFrom
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferFrom
 */
export declare const MsgTransferFrom: {
    typeUrl: string;
    encode(message: MsgTransferFrom, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgTransferFrom;
    fromPartial(object: DeepPartial<MsgTransferFrom>): MsgTransferFrom;
};
/**
 * @name MsgTransferFromResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferFromResponse
 */
export declare const MsgTransferFromResponse: {
    typeUrl: string;
    encode(_: MsgTransferFromResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgTransferFromResponse;
    fromPartial(_: DeepPartial<MsgTransferFromResponse>): MsgTransferFromResponse;
};
/**
 * @name MsgPauseToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgPauseToken
 */
export declare const MsgPauseToken: {
    typeUrl: string;
    encode(message: MsgPauseToken, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseToken;
    fromPartial(object: DeepPartial<MsgPauseToken>): MsgPauseToken;
};
/**
 * @name MsgPauseTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgPauseTokenResponse
 */
export declare const MsgPauseTokenResponse: {
    typeUrl: string;
    encode(_: MsgPauseTokenResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseTokenResponse;
    fromPartial(_: DeepPartial<MsgPauseTokenResponse>): MsgPauseTokenResponse;
};
/**
 * @name MsgUnpauseToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnpauseToken
 */
export declare const MsgUnpauseToken: {
    typeUrl: string;
    encode(message: MsgUnpauseToken, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUnpauseToken;
    fromPartial(object: DeepPartial<MsgUnpauseToken>): MsgUnpauseToken;
};
/**
 * @name MsgUnpauseTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnpauseTokenResponse
 */
export declare const MsgUnpauseTokenResponse: {
    typeUrl: string;
    encode(_: MsgUnpauseTokenResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUnpauseTokenResponse;
    fromPartial(_: DeepPartial<MsgUnpauseTokenResponse>): MsgUnpauseTokenResponse;
};
/**
 * @name MsgDelegatePower
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgDelegatePower
 */
export declare const MsgDelegatePower: {
    typeUrl: string;
    encode(message: MsgDelegatePower, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgDelegatePower;
    fromPartial(object: DeepPartial<MsgDelegatePower>): MsgDelegatePower;
};
/**
 * @name MsgDelegatePowerResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgDelegatePowerResponse
 */
export declare const MsgDelegatePowerResponse: {
    typeUrl: string;
    encode(_: MsgDelegatePowerResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgDelegatePowerResponse;
    fromPartial(_: DeepPartial<MsgDelegatePowerResponse>): MsgDelegatePowerResponse;
};
/**
 * @name MsgUndelegatePower
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUndelegatePower
 */
export declare const MsgUndelegatePower: {
    typeUrl: string;
    encode(message: MsgUndelegatePower, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUndelegatePower;
    fromPartial(object: DeepPartial<MsgUndelegatePower>): MsgUndelegatePower;
};
/**
 * @name MsgUndelegatePowerResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUndelegatePowerResponse
 */
export declare const MsgUndelegatePowerResponse: {
    typeUrl: string;
    encode(_: MsgUndelegatePowerResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUndelegatePowerResponse;
    fromPartial(_: DeepPartial<MsgUndelegatePowerResponse>): MsgUndelegatePowerResponse;
};
/**
 * @name MsgWrapToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgWrapToken
 */
export declare const MsgWrapToken: {
    typeUrl: string;
    encode(message: MsgWrapToken, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgWrapToken;
    fromPartial(object: DeepPartial<MsgWrapToken>): MsgWrapToken;
};
/**
 * @name MsgWrapTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgWrapTokenResponse
 */
export declare const MsgWrapTokenResponse: {
    typeUrl: string;
    encode(message: MsgWrapTokenResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgWrapTokenResponse;
    fromPartial(object: DeepPartial<MsgWrapTokenResponse>): MsgWrapTokenResponse;
};
/**
 * @name MsgUnwrapToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnwrapToken
 */
export declare const MsgUnwrapToken: {
    typeUrl: string;
    encode(message: MsgUnwrapToken, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUnwrapToken;
    fromPartial(object: DeepPartial<MsgUnwrapToken>): MsgUnwrapToken;
};
/**
 * @name MsgUnwrapTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnwrapTokenResponse
 */
export declare const MsgUnwrapTokenResponse: {
    typeUrl: string;
    encode(message: MsgUnwrapTokenResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUnwrapTokenResponse;
    fromPartial(object: DeepPartial<MsgUnwrapTokenResponse>): MsgUnwrapTokenResponse;
};
/**
 * @name MsgCreateEmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateEmissionPeriod
 */
export declare const MsgCreateEmissionPeriod: {
    typeUrl: string;
    encode(message: MsgCreateEmissionPeriod, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateEmissionPeriod;
    fromPartial(object: DeepPartial<MsgCreateEmissionPeriod>): MsgCreateEmissionPeriod;
};
/**
 * @name MsgCreateEmissionPeriodResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateEmissionPeriodResponse
 */
export declare const MsgCreateEmissionPeriodResponse: {
    typeUrl: string;
    encode(message: MsgCreateEmissionPeriodResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateEmissionPeriodResponse;
    fromPartial(object: DeepPartial<MsgCreateEmissionPeriodResponse>): MsgCreateEmissionPeriodResponse;
};
/**
 * @name MsgCancelEmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCancelEmissionPeriod
 */
export declare const MsgCancelEmissionPeriod: {
    typeUrl: string;
    encode(message: MsgCancelEmissionPeriod, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelEmissionPeriod;
    fromPartial(object: DeepPartial<MsgCancelEmissionPeriod>): MsgCancelEmissionPeriod;
};
/**
 * @name MsgCancelEmissionPeriodResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCancelEmissionPeriodResponse
 */
export declare const MsgCancelEmissionPeriodResponse: {
    typeUrl: string;
    encode(_: MsgCancelEmissionPeriodResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelEmissionPeriodResponse;
    fromPartial(_: DeepPartial<MsgCancelEmissionPeriodResponse>): MsgCancelEmissionPeriodResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
