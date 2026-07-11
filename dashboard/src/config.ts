export const CHAIN_ID = "zerone-1";
export const CHAIN_NAME = "Zerone Mainnet";
export const DENOM = "uzrn";
export const DISPLAY_DENOM = "ZRN";
export const DECIMALS = 6;
export const HARD_CAP_ZRN = 222_222_222;

export const RPC_ENDPOINT = `${window.location.origin}/api/rpc`;
export const REST_ENDPOINT = `${window.location.origin}/api/rest`;

export const KEPLR_CHAIN_INFO = {
  chainId: CHAIN_ID,
  chainName: CHAIN_NAME,
  rpc: RPC_ENDPOINT,
  rest: REST_ENDPOINT,
  bip44: { coinType: 118 },
  bech32Config: {
    bech32PrefixAccAddr: "zrn",
    bech32PrefixAccPub: "zrnpub",
    bech32PrefixValAddr: "zrnvaloper",
    bech32PrefixValPub: "zrnvaloperpub",
    bech32PrefixConsAddr: "zrnvalcons",
    bech32PrefixConsPub: "zrnvalconspub",
  },
  currencies: [
    {
      coinDenom: DISPLAY_DENOM,
      coinMinimalDenom: DENOM,
      coinDecimals: DECIMALS,
    },
  ],
  feeCurrencies: [
    {
      coinDenom: DISPLAY_DENOM,
      coinMinimalDenom: DENOM,
      coinDecimals: DECIMALS,
      gasPriceStep: { low: 1, average: 1, high: 1.2 },
    },
  ],
  stakeCurrency: {
    coinDenom: DISPLAY_DENOM,
    coinMinimalDenom: DENOM,
    coinDecimals: DECIMALS,
  },
  features: ["ibc-transfer"],
};
