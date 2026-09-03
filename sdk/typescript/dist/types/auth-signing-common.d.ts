export declare const U32_MAX = 4294967295;
export declare function encodeChainId(chainId: string): Uint8Array;
export declare function encodeText(value: string, label: string): Uint8Array;
export declare function validateZeroneAddress(address: string): void;
export declare function writeUint32(output: Uint8Array, offset: number, value: number): number;
