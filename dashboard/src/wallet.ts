import {
  isOfflineDirectSigner,
  type EncodeObject,
  type OfflineSigner,
} from "@cosmjs/proto-signing";
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
  getFeeGrant,
  getFeeGrantsByGrantee,
  getWalletBalance,
  type AccountIdentifier,
} from "./api";
import {
  BANK_SEND_TYPE_URL,
  createBoundedGrantMessage,
  createRevokeGrantMessage,
  feeGrantAllowsMessage,
  type FeeGrantAllowance,
} from "./feegrant";
import {
  CHAIN_ID,
  DECIMALS,
  DENOM,
  FEEGRANT_SPONSORSHIP_ENABLED,
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
  incomingFeeGrants: FeeGrantAllowance[] | null;
}

const BANK_SEND_GAS = 200_000;
const BANK_SEND_FEE_UZRN = BigInt(BANK_SEND_GAS);
const FEE_GRANT_GAS = 250_000;
const FEE_REVOKE_GAS = 200_000;
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

async function loadFeeGrants(address: string): Promise<
  Pick<WalletState, "incomingFeeGrants">
> {
  try {
    return { incomingFeeGrants: await getFeeGrantsByGrantee(address) };
  } catch {
    return { incomingFeeGrants: null };
  }
}

export async function connectWallet(): Promise<WalletState> {
  const keplr = requireKeplr();
  await keplr.experimentalSuggestChain(KEPLR_CHAIN_INFO);
  await keplr.enable(CHAIN_ID);
  const key = await keplr.getKey(CHAIN_ID);
  const accountId = accountIdFor(key.bech32Address);
  const [balanceUzrn, identifier, feeGrants] = await Promise.all([
    getWalletBalance(key.bech32Address),
    getAccountIdentifier(key.bech32Address),
    loadFeeGrants(key.bech32Address),
  ]);
  if (identifier) {
    requireMatchingIdentifier(key.bech32Address, accountId, identifier);
  }
  return {
    name: key.name,
    address: key.bech32Address,
    accountId,
    balanceUzrn,
    ...feeGrants,
    ...(identifier ? identityFields(identifier) : {}),
  };
}

export async function refreshWallet(wallet: WalletState): Promise<WalletState> {
  const [balanceUzrn, identifier, feeGrants] = await Promise.all([
    getWalletBalance(wallet.address),
    getAccountIdentifier(wallet.address),
    loadFeeGrants(wallet.address),
  ]);
  if (identifier) {
    requireMatchingIdentifier(wallet.address, wallet.accountId, identifier);
  }
  const refreshed: WalletState = {
    name: wallet.name,
    address: wallet.address,
    accountId: wallet.accountId,
    balanceUzrn,
    ...feeGrants,
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
  feeGranter?: string,
): Promise<{ transactionHash: string; gasUsed: bigint; gasWanted: bigint }> {
  requireKeplr();
  if (wallet.frozen === true) {
    throw new Error("This account is frozen on-chain and cannot send ZRN.");
  }
  const microAmount = displayToMicro(amount);
  accountIdFor(recipient);
  if (memo.length > 256) throw new Error("Memo must be 256 characters or fewer.");

  let normalizedFeeGranter: string | undefined;
  if (feeGranter) {
    if (!FEEGRANT_SPONSORSHIP_ENABLED) {
      throw new Error(
        "Sponsored sends are waiting for the live validator freeze guard.",
      );
    }
    accountIdFor(feeGranter);
    if (feeGranter === wallet.address) {
      throw new Error("Choose your own balance or a different fee sponsor.");
    }
    const grant = await getFeeGrant(feeGranter, wallet.address);
    if (
      grant === null ||
      grant.granter !== feeGranter ||
      grant.grantee !== wallet.address ||
      !feeGrantAllowsMessage(
        grant,
        BANK_SEND_TYPE_URL,
        BANK_SEND_FEE_UZRN,
      )
    ) {
      throw new Error(
        "That fee grant is no longer available for this bank send.",
      );
    }
    normalizedFeeGranter = feeGranter;
  }

  const freshBalanceUzrn = await getWalletBalance(wallet.address);
  const requiredBalance =
    BigInt(microAmount) +
    (normalizedFeeGranter ? 0n : BANK_SEND_FEE_UZRN);
  if (requiredBalance > BigInt(freshBalanceUzrn)) {
    throw new Error(
      normalizedFeeGranter
        ? "The available balance is below the amount being sent."
        : "Leave at least 0.20 ZRN available for the network fee.",
    );
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
      const baseFee = calculateFee(BANK_SEND_GAS, gasPrice);
      result = await client.sendTokens(
        wallet.address,
        recipient,
        [coin(microAmount, DENOM)],
        normalizedFeeGranter
          ? { ...baseFee, granter: normalizedFeeGranter }
          : baseFee,
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

async function broadcastFeeGrantMessage(
  wallet: WalletState,
  message: EncodeObject,
  gas: number,
): Promise<{ transactionHash: string; gasUsed: bigint; gasWanted: bigint }> {
  requireKeplr();
  if (wallet.frozen === true) {
    throw new Error("This account is frozen on-chain and cannot change fee grants.");
  }
  const freshBalanceUzrn = await getWalletBalance(wallet.address);
  if (BigInt(freshBalanceUzrn) < BigInt(gas)) {
    throw new Error(
      `Leave at least ${(gas / 1_000_000).toFixed(2)} ZRN available for the network fee.`,
    );
  }

  const signer = window.getOfflineSigner!(CHAIN_ID);
  if (!isOfflineDirectSigner(signer)) {
    throw new Error(
      "This feegrant transaction requires a wallet with protobuf direct signing.",
    );
  }
  const signerAccounts = await signer.getAccounts();
  if (!signerAccounts.some((account) => account.address === wallet.address)) {
    throw new Error("Keplr changed accounts. Reconnect the wallet before signing.");
  }
  const gasPrice = GasPrice.fromString(`1${DENOM}`);
  const client = await SigningStargateClient.connectWithSigner(
    RPC_ENDPOINT,
    signer,
    { gasPrice },
  );

  try {
    const connectedChainId = await client.getChainId();
    if (connectedChainId !== CHAIN_ID) {
      throw new Error(`Wallet RPC reported ${connectedChainId}, expected ${CHAIN_ID}.`);
    }
    let result;
    try {
      result = await client.signAndBroadcast(
        wallet.address,
        [message],
        calculateFee(gas, gasPrice),
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

export async function grantFeeAllowance(
  wallet: WalletState,
  input: {
    grantee: string;
    spendLimitZrn: string;
    expiration: Date;
    allowedMessages: readonly string[];
  },
): Promise<{ transactionHash: string; gasUsed: bigint; gasWanted: bigint }> {
  if (!FEEGRANT_SPONSORSHIP_ENABLED) {
    throw new Error(
      "Fee grant creation is waiting for the live validator freeze guard.",
    );
  }
  return broadcastFeeGrantMessage(
    wallet,
    createBoundedGrantMessage({
      granter: wallet.address,
      ...input,
    }),
    FEE_GRANT_GAS,
  );
}

export async function revokeFeeAllowance(
  wallet: WalletState,
  grantee: string,
): Promise<{ transactionHash: string; gasUsed: bigint; gasWanted: bigint }> {
  const existing = await getFeeGrant(wallet.address, grantee);
  if (
    existing === null ||
    existing.granter !== wallet.address ||
    existing.grantee !== grantee
  ) {
    throw new Error("No fee grant exists for that grantee.");
  }
  return broadcastFeeGrantMessage(
    wallet,
    createRevokeGrantMessage(wallet.address, grantee),
    FEE_REVOKE_GAS,
  );
}
