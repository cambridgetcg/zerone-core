declare const caip2Brand: unique symbol;
declare const caip10Brand: unique symbol;
declare const cosmosChainBrand: unique symbol;
declare const zeroneChainBrand: unique symbol;
declare const zeroneAccountBrand: unique symbol;
declare const zeroneNetworkBrand: unique symbol;
declare const zeroneDidBrand: unique symbol;
export type Caip2Id = string & {
    readonly [caip2Brand]: true;
};
export type Caip10Id = string & {
    readonly [caip10Brand]: true;
};
export type CosmosChainId = Caip2Id & {
    readonly [cosmosChainBrand]: true;
};
export type ZeroneChainId = CosmosChainId & {
    readonly [zeroneChainBrand]: true;
};
export type ZeroneAccountId = Caip10Id & {
    readonly [zeroneAccountBrand]: true;
};
export type ZeroneDidRef = string & {
    readonly [zeroneDidBrand]: true;
};
/**
 * An application-owned declaration that a Tendermint chain is a Zerone
 * network. Construct it once from trusted network configuration rather than
 * inferring chain semantics from an account's Bech32 prefix.
 */
export interface ZeroneNetwork {
    readonly rawChainId: string;
    readonly chainId: ZeroneChainId;
    readonly accountPrefix: "zrn";
    readonly accountBytes: 20;
    readonly [zeroneNetworkBrand]: true;
}
export interface Caip2Parts {
    readonly id: Caip2Id;
    readonly namespace: string;
    readonly reference: string;
}
export interface Caip10Parts {
    readonly id: Caip10Id;
    readonly chainId: Caip2Id;
    readonly namespace: string;
    readonly reference: string;
    readonly accountAddress: string;
}
export interface ZeroneIdentityRef {
    readonly accountId: ZeroneAccountId;
    readonly did?: ZeroneDidRef;
}
export type CaipErrorCode = "INVALID_CAIP2" | "INVALID_CAIP10" | "NOT_COSMOS" | "INVALID_COSMOS_REFERENCE" | "INVALID_BECH32" | "NON_CANONICAL_ADDRESS" | "WRONG_HRP" | "WRONG_ACCOUNT_LENGTH" | "INVALID_DID_ZRN";
export declare class CaipError extends Error {
    readonly code: CaipErrorCode;
    constructor(code: CaipErrorCode, message: string);
}
export declare function parseCaip2(value: string): Caip2Parts;
export declare function formatCaip2(namespace: string, reference: string): Caip2Id;
export declare function parseCaip10(value: string): Caip10Parts;
export declare function formatCaip10(chainId: Caip2Id, accountAddress: string): Caip10Id;
export declare function cosmosChainId(tendermintChainId: string): CosmosChainId;
export declare function parseCosmosChainId(value: string): CosmosChainId;
export declare function defineZeroneNetwork(rawChainId: string): ZeroneNetwork;
export declare function zeroneAccountId(network: ZeroneNetwork, address: string): ZeroneAccountId;
/**
 * Validates the method-specific identifier accepted by Zerone x/auth today.
 * This does not claim that did:zrn is a published W3C DID method.
 */
export declare function asExistingZeroneDid(value: string): ZeroneDidRef;
export {};
