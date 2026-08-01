import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Registry } from "@cosmjs/proto-signing";
import { defaultRegistryTypes } from "@cosmjs/stargate";
import {
  AllowedMsgAllowance,
  BasicAllowance,
} from "cosmjs-types/cosmos/feegrant/v1beta1/feegrant";
import {
  MsgGrantAllowance,
  MsgRevokeAllowance,
} from "cosmjs-types/cosmos/feegrant/v1beta1/tx";
import { CaipError, defineZeroneNetwork } from "../src/caip";
import {
  FeeGrantError,
  ZERONE_ONBOARDING_MESSAGE_TYPE_URLS,
  makeBoundedFeeGrant,
  makeRevokeFeeGrant,
  makeSponsoredFee,
  type BoundedFeeGrantInput,
} from "../src/feegrant";

const NETWORK = defineZeroneNetwork("zerone-1");
const GRANTER = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";
const GRANTEE = "zrn1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5s75sh2";
const EXPIRATION = new Date("2099-06-01T12:34:56.123Z");

function validGrant(
  overrides: Partial<BoundedFeeGrantInput> = {},
): BoundedFeeGrantInput {
  return {
    network: NETWORK,
    granter: GRANTER,
    grantee: GRANTEE,
    spendLimit: [{ denom: "uzrn", amount: "100000" }],
    expiration: EXPIRATION,
    allowedMessageTypeUrls: ["/zerone.claiming_pot.v1.MsgClaim"],
    ...overrides,
  };
}

function assertFeeGrantError(
  operation: () => unknown,
  code: InstanceType<typeof FeeGrantError>["code"],
): void {
  assert.throws(
    operation,
    (error: unknown) => error instanceof FeeGrantError && error.code === code,
  );
}

describe("bounded Zerone fee grants", () => {
  it("encodes BasicAllowance inside AllowedMsgAllowance", () => {
    const message = makeBoundedFeeGrant(
      validGrant({
        spendLimit: [
          { denom: "uzrn", amount: "100000" },
          {
            denom:
              "ibc/27394FB092D2A322A54F01F5C0A8EE993A902FFD3D1FECB480A1A25537A0B2A2",
            amount: "20",
          },
        ],
        allowedMessageTypeUrls: ["/zerone.claiming_pot.v1.MsgClaim"],
      }),
    );

    assert.equal(message.typeUrl, MsgGrantAllowance.typeUrl);
    const grantWire = new Registry(defaultRegistryTypes).encode(message);
    const grant = MsgGrantAllowance.decode(grantWire);
    assert.equal(grant.granter, GRANTER);
    assert.equal(grant.grantee, GRANTEE);
    assert.equal(grant.allowance?.typeUrl, AllowedMsgAllowance.typeUrl);

    const allowed = AllowedMsgAllowance.decode(
      assertDefined(grant.allowance).value,
    );
    assert.deepEqual(
      allowed.allowedMessages,
      ZERONE_ONBOARDING_MESSAGE_TYPE_URLS,
    );
    assert.equal(allowed.allowance?.typeUrl, BasicAllowance.typeUrl);

    const basic = BasicAllowance.decode(
      assertDefined(allowed.allowance).value,
    );
    assert.deepEqual(basic.spendLimit, [
      {
        denom:
          "ibc/27394FB092D2A322A54F01F5C0A8EE993A902FFD3D1FECB480A1A25537A0B2A2",
        amount: "20",
      },
      { denom: "uzrn", amount: "100000" },
    ]);
    assert.deepEqual(basic.expiration, {
      seconds: BigInt(Math.floor(EXPIRATION.getTime() / 1_000)),
      nanos: 123_000_000,
    });
  });

  it("builds an encodable revoke message", () => {
    const message = makeRevokeFeeGrant({
      network: NETWORK,
      granter: GRANTER,
      grantee: GRANTEE,
    });
    assert.equal(message.typeUrl, MsgRevokeAllowance.typeUrl);
    assert.deepEqual(
      MsgRevokeAllowance.decode(
        new Registry(defaultRegistryTypes).encode(message),
      ),
      { granter: GRANTER, grantee: GRANTEE },
    );
  });

  it("validates both Zerone parties and rejects self-grants", () => {
    assert.throws(
      () =>
        makeBoundedFeeGrant(
          validGrant({ grantee: "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a" }),
        ),
      CaipError,
    );
    assertFeeGrantError(
      () => makeBoundedFeeGrant(validGrant({ grantee: GRANTER })),
      "SELF_GRANT",
    );
    assertFeeGrantError(
      () =>
        makeRevokeFeeGrant({
          network: NETWORK,
          granter: GRANTER,
          grantee: GRANTER,
        }),
      "SELF_GRANT",
    );
  });

  it("rejects empty, invalid, noncanonical, and duplicate spend coins", () => {
    const cases: ReadonlyArray<{
      readonly spendLimit: BoundedFeeGrantInput["spendLimit"];
      readonly code: InstanceType<typeof FeeGrantError>["code"];
    }> = [
      { spendLimit: [], code: "EMPTY_SPEND_LIMIT" },
      {
        spendLimit: [{ denom: "u", amount: "1" }],
        code: "INVALID_COIN",
      },
      {
        spendLimit: [{ denom: "uzrn", amount: "0" }],
        code: "INVALID_COIN",
      },
      {
        spendLimit: [{ denom: "uzrn", amount: "01" }],
        code: "INVALID_COIN",
      },
      {
        spendLimit: [
          { denom: "uzrn", amount: "1" },
          { denom: "uzrn", amount: "2" },
        ],
        code: "DUPLICATE_DENOM",
      },
    ];

    for (const { spendLimit, code } of cases) {
      assertFeeGrantError(
        () => makeBoundedFeeGrant(validGrant({ spendLimit })),
        code,
      );
    }
  });

  it("requires a valid future expiry", () => {
    assertFeeGrantError(
      () =>
        makeBoundedFeeGrant(
          validGrant({ expiration: new Date("2000-01-01T00:00:00Z") }),
        ),
      "EXPIRED_ALLOWANCE",
    );
    assertFeeGrantError(
      () =>
        makeBoundedFeeGrant(
          validGrant({ expiration: new Date("invalid") }),
        ),
      "INVALID_EXPIRATION",
    );
  });

  it("requires unique exact message URLs", () => {
    assertFeeGrantError(
      () =>
        makeBoundedFeeGrant(
          validGrant({ allowedMessageTypeUrls: [] }),
        ),
      "EMPTY_ALLOWED_MESSAGES",
    );
    assertFeeGrantError(
      () =>
        makeBoundedFeeGrant(
          validGrant({
            allowedMessageTypeUrls: ["/cosmos.bank.v1beta1.Msg*"],
          }),
        ),
      "INVALID_MESSAGE_TYPE_URL",
    );
    assertFeeGrantError(
      () =>
        makeBoundedFeeGrant(
          validGrant({
            allowedMessageTypeUrls: [
              "/zerone.claiming_pot.v1.MsgClaim",
              "/zerone.claiming_pot.v1.MsgClaim",
            ],
          }),
        ),
      "DUPLICATE_MESSAGE_TYPE_URL",
    );
  });

  it("rejects every message outside the explicit onboarding allowlist", () => {
    const unapprovedTypeUrls = [
      "/zerone.auth.v1.MsgRegisterAccount",
      "/zerone.emergency.v1.MsgPause",
      "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "/cosmos.gov.v1.MsgVote",
      "/cosmos.params.v1beta1.MsgUpdateParams",
      "/cosmwasm.wasm.v1.MsgUpdateAdmin",
      "/cosmos.authz.v1beta1.MsgExec",
      "/zerone.auth.v1.MsgFreezeAccount",
      "/zerone.auth.v1.MsgUnfreezeAccount",
      "/zerone.auth.v1.MsgRotateKey",
      "/zerone.knowledge.v1.MsgPauseModule",
      "/zerone.creed.v1.MsgUpdateCouncilMember",
      "/zerone.substrate_bridge.v1.MsgSuspendAdapter",
      "/zerone.tokens.v1.MsgPauseToken",
    ];

    for (const typeUrl of unapprovedTypeUrls) {
      assertFeeGrantError(
        () =>
          makeBoundedFeeGrant(
            validGrant({ allowedMessageTypeUrls: [typeUrl] }),
          ),
        "UNAPPROVED_MESSAGE_TYPE_URL",
      );
    }
  });
});

describe("sponsored CosmJS fees", () => {
  it("adds a validated granter and canonicalizes the positive fee coins", () => {
    assert.deepEqual(
      makeSponsoredFee({
        network: NETWORK,
        granter: GRANTER,
        amount: [
          { denom: "uzrn", amount: "2500" },
          { denom: "aaa", amount: "1" },
        ],
        gas: "200000",
      }),
      {
        amount: [
          { denom: "aaa", amount: "1" },
          { denom: "uzrn", amount: "2500" },
        ],
        gas: "200000",
        granter: GRANTER,
      },
    );
    assert.equal(
      makeSponsoredFee({
        network: NETWORK,
        granter: GRANTER,
        amount: [{ denom: "uzrn", amount: "1" }],
        gas: "9007199254740991",
      }).gas,
      "9007199254740991",
    );
  });

  it("rejects invalid fees and granter addresses", () => {
    assertFeeGrantError(
      () =>
        makeSponsoredFee({
          network: NETWORK,
          granter: GRANTER,
          amount: [],
          gas: "200000",
        }),
      "INVALID_COIN",
    );
    for (const gas of [
      "0",
      "01",
      "-1",
      "9007199254740992",
      "18446744073709551615",
    ]) {
      assertFeeGrantError(
        () =>
          makeSponsoredFee({
            network: NETWORK,
            granter: GRANTER,
            amount: [{ denom: "uzrn", amount: "1" }],
            gas,
          }),
        "INVALID_GAS",
      );
    }
    assert.throws(
      () =>
        makeSponsoredFee({
          network: NETWORK,
          granter: GRANTER.toUpperCase(),
          amount: [{ denom: "uzrn", amount: "1" }],
          gas: "1",
        }),
      CaipError,
    );
  });
});

function assertDefined<T>(value: T | undefined): T {
  assert.notEqual(value, undefined);
  return value as T;
}
