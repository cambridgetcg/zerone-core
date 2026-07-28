import type { OfflineSigner } from "@cosmjs/proto-signing";
import {
  calculateFee,
  coin,
  GasPrice,
  SigningStargateClient,
} from "@cosmjs/stargate";
import {
  asExistingZeroneDid,
  defineZeroneNetwork,
  zeroneAccountId,
  type ZeroneAccountId,
  type ZeroneDidRef,
} from "@zerone-chain/sdk/caip";
import {
  getAccountIdentifier,
  getWalletBalance,
  type AccountIdentifier,
} from "./api";
import {
  CHAIN_ID,
  DECIMALS,
  DENOM,
  KEPLR_CHAIN_INFO,
  RPC_ENDPOINT,
} from "./config";

interface KeplrKey {
  name: string;
  bech32Address: string;
}

interface KeplrApi {
  experimentalSuggestChain(chainInfo: typeof KEPLR_CHAIN_INFO): Promise<void>;
  enable(chainId: string): Promise<void>;
  getKey(chainId: string): Promise<KeplrKey>;
}

declare global {
  interface Window {
    keplr?: KeplrApi;
    getOfflineSigner?: (chainId: string) => OfflineSigner;
  }
}

export interface WalletState {
  name: string;
  address: string;
  accountId: ZeroneAccountId;
  balanceUzrn: string;
  did?: ZeroneDidRef;
  accountType?: string;
  frozen?: boolean;
  createdAtBlock?: string;
}

const BANK_SEND_GAS = 200_000;
const BANK_SEND_FEE_UZRN = BigInt(BANK_SEND_GAS);
const ZERONE_NETWORK = defineZeroneNetwork(CHAIN_ID);

export class SubmittedTransactionError extends Error {
  readonly transactionHash: string;

  constructor(message: string, transactionHash: string) {
    super(message);
    this.name = "SubmittedTransactionError";
    this.transactionHash = transactionHash;
  }
}

function requireKeplr(): KeplrApi {
  if (!window.keplr || !window.getOfflineSigner) {
    throw new Error("Keplr is not installed. Add the extension, then try again.");
  }
  return window.keplr;
}

function accountIdFor(address: string): ZeroneAccountId {
  try {
    return zeroneAccountId(ZERONE_NETWORK, address);
  } catch {
    throw new Error("Enter a valid zrn1… mainnet address.");
  }
}

function requireMatchingIdentifier(
  expectedAddress: string,
  expectedAccountId: ZeroneAccountId,
  identifier: AccountIdentifier,
): void {
  if (
    identifier.namespace !== "cosmos" ||
    `${identifier.namespace}:${identifier.reference}` !== ZERONE_NETWORK.chainId ||
    identifier.rawChainId !== CHAIN_ID ||
    identifier.address !== expectedAddress ||
    identifier.accountId !== expectedAccountId
  ) {
    throw new Error(
      "Mainnet returned an account identity that does not match the connected wallet.",
    );
  }
}

function identityFields(identifier: AccountIdentifier): Pick<
  WalletState,
  "did" | "accountType" | "frozen" | "createdAtBlock"
> {
  return {
    did: asExistingZeroneDid(identifier.did),
    accountType: identifier.accountType,
    frozen: identifier.frozen,
    createdAtBlock: identifier.createdAtBlock,
  };
}

export async function connectWallet(): Promise<WalletState> {
  const keplr = requireKeplr();
  await keplr.experimentalSuggestChain(KEPLR_CHAIN_INFO);
  await keplr.enable(CHAIN_ID);
  const key = await keplr.getKey(CHAIN_ID);
  const accountId = accountIdFor(key.bech32Address);
  const [balanceUzrn, identifier] = await Promise.all([
    getWalletBalance(key.bech32Address),
    getAccountIdentifier(key.bech32Address),
  ]);
  if (identifier) {
    requireMatchingIdentifier(key.bech32Address, accountId, identifier);
  }
  return {
    name: key.name,
    address: key.bech32Address,
    accountId,
    balanceUzrn,
    ...(identifier ? identityFields(identifier) : {}),
  };
}

export async function refreshWallet(wallet: WalletState): Promise<WalletState> {
  const [balanceUzrn, identifier] = await Promise.all([
    getWalletBalance(wallet.address),
    getAccountIdentifier(wallet.address),
  ]);
  if (identifier) {
    requireMatchingIdentifier(wallet.address, wallet.accountId, identifier);
  }
  const refreshed: WalletState = {
    name: wallet.name,
    address: wallet.address,
    accountId: wallet.accountId,
    balanceUzrn,
  };
  return identifier
    ? { ...refreshed, ...identityFields(identifier) }
    : refreshed;
}

function displayToMicro(amount: string): string {
  const normalized = amount.trim();
  if (!/^\d+(?:\.\d{0,6})?$/.test(normalized)) {
    throw new Error(`Use no more than ${DECIMALS} decimal places.`);
  }
  const [whole = "0", fraction = ""] = normalized.split(".");
  const micro = BigInt(whole) * 10n ** BigInt(DECIMALS) + BigInt(fraction.padEnd(DECIMALS, "0"));
  if (micro <= 0n) throw new Error("Amount must be greater than zero.");
  return micro.toString();
}

export async function sendZrn(
  wallet: WalletState,
  recipient: string,
  amount: string,
  memo: string,
): Promise<{ transactionHash: string; gasUsed: bigint; gasWanted: bigint }> {
  requireKeplr();
  if (wallet.frozen === true) {
    throw new Error("This account is frozen on-chain and cannot send ZRN.");
  }
  const microAmount = displayToMicro(amount);
  accountIdFor(recipient);
  if (memo.length > 256) throw new Error("Memo must be 256 characters or fewer.");

  const freshBalanceUzrn = await getWalletBalance(wallet.address);
  if (BigInt(microAmount) + BANK_SEND_FEE_UZRN > BigInt(freshBalanceUzrn)) {
    throw new Error("Leave at least 0.20 ZRN available for the network fee.");
  }

  const signer = window.getOfflineSigner!(CHAIN_ID);
  const signerAccounts = await signer.getAccounts();
  if (!signerAccounts.some((account) => account.address === wallet.address)) {
    throw new Error("Keplr changed accounts. Reconnect the wallet before sending.");
  }
  const gasPrice = GasPrice.fromString(`1${DENOM}`);
  const client = await SigningStargateClient.connectWithSigner(RPC_ENDPOINT, signer, {
    gasPrice,
  });

  try {
    const connectedChainId = await client.getChainId();
    if (connectedChainId !== CHAIN_ID) {
      throw new Error(`Wallet RPC reported ${connectedChainId}, expected ${CHAIN_ID}.`);
    }

    let result;
    try {
      result = await client.sendTokens(
        wallet.address,
        recipient,
        [coin(microAmount, DENOM)],
        calculateFee(BANK_SEND_GAS, gasPrice),
        memo.trim(),
      );
    } catch (error) {
      const txId =
        typeof error === "object" && error !== null && "txId" in error
          ? String(error.txId)
          : "";
      if (txId) {
        throw new SubmittedTransactionError(
          "Transaction was broadcast but confirmation is still pending.",
          txId,
        );
      }
      throw error;
    }
    if (result.code !== 0) {
      throw new SubmittedTransactionError(
        result.rawLog || `Transaction was included but failed with code ${result.code}.`,
        result.transactionHash,
      );
    }
    return {
      transactionHash: result.transactionHash,
      gasUsed: result.gasUsed,
      gasWanted: result.gasWanted,
    };
  } finally {
    client.disconnect();
  }
}
