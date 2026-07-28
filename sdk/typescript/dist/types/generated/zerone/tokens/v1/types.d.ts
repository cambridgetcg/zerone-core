import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * TokenFeatures controls which operations are enabled for a token.
 * @name TokenFeatures
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.TokenFeatures
 */
export interface TokenFeatures {
    mintable: boolean;
    burnable: boolean;
    pausable: boolean;
    wrappable: boolean;
}
/**
 * TokenDefinition is a ZRN-20 secondary token's on-chain configuration.
 * @name TokenDefinition
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.TokenDefinition
 */
export interface TokenDefinition {
    id: string;
    /**
     * address that created the token (= mint authority)
     */
    creator: string;
    name: string;
    /**
     * 1-16 uppercase alphanumeric
     */
    symbol: string;
    decimals: number;
    /**
     * bigint as string
     */
    totalSupply: string;
    /**
     * "0" = unlimited
     */
    maxSupply: string;
    features?: TokenFeatures;
    paused: boolean;
    /**
     * block height
     */
    createdAt: bigint;
}
/**
 * EmissionPeriod defines a scheduled token emission (replaces LP pools).
 * @name EmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.EmissionPeriod
 */
export interface EmissionPeriod {
    id: string;
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
    active: boolean;
    /**
     * running total emitted
     */
    totalEmitted: string;
    /**
     * governance authority that created it
     */
    creator: string;
}
/**
 * TokenFeatures controls which operations are enabled for a token.
 * @name TokenFeatures
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.TokenFeatures
 */
export declare const TokenFeatures: {
    typeUrl: string;
    encode(message: TokenFeatures, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TokenFeatures;
    fromPartial(object: DeepPartial<TokenFeatures>): TokenFeatures;
};
/**
 * TokenDefinition is a ZRN-20 secondary token's on-chain configuration.
 * @name TokenDefinition
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.TokenDefinition
 */
export declare const TokenDefinition: {
    typeUrl: string;
    encode(message: TokenDefinition, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TokenDefinition;
    fromPartial(object: DeepPartial<TokenDefinition>): TokenDefinition;
};
/**
 * EmissionPeriod defines a scheduled token emission (replaces LP pools).
 * @name EmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.EmissionPeriod
 */
export declare const EmissionPeriod: {
    typeUrl: string;
    encode(message: EmissionPeriod, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmissionPeriod;
    fromPartial(object: DeepPartial<EmissionPeriod>): EmissionPeriod;
};
