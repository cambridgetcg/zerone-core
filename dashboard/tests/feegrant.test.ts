import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  AllowedMsgAllowance,
  BasicAllowance,
} from "cosmjs-types/cosmos/feegrant/v1beta1/feegrant";
import type {
  MsgGrantAllowance,
  MsgRevokeAllowance,
} from "cosmjs-types/cosmos/feegrant/v1beta1/tx";
import {
  BANK_SEND_TYPE_URL,
  CLAIM_TYPE_URL,
  createBoundedGrantMessage,
  createRevokeGrantMessage,
  feeGrantAllowsMessage,
  isMissingFeeGrantError,
  parseFeeGrantPage,
} from "../src/feegrant";

const GRANTER = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";
const GRANTEE = "zrn100mxrvv5chhhrj0yd9y4q8354z4edm42mukf5r";
const BASIC_TYPE = "/cosmos.feegrant.v1beta1.BasicAllowance";
const PERIODIC_TYPE = "/cosmos.feegrant.v1beta1.PeriodicAllowance";
const ALLOWED_TYPE = "/cosmos.feegrant.v1beta1.AllowedMsgAllowance";
const NOW = Date.parse("2026-07-29T12:00:00Z");

function basicAllowance(
  amount = "1000000",
  expiration = "2026-08-05T12:00:00Z",
): Record<string, unknown> {
  return {
    "@type": BASIC_TYPE,
    spend_limit: [{ denom: "uzrn", amount }],
    expiration,
  };
}

function responseWith(allowance: unknown): unknown {
  return {
    allowances: [{ granter: GRANTER, grantee: GRANTEE, allowance }],
    pagination: { next_key: null, total: "1" },
  };
}

describe("feegrant REST parsing", () => {
  it("recognizes only the live keeper's exact missing-grant error", () => {
    const missing = {
      code: 13,
      message: "fee-grant not found: not found",
      details: [],
    };
    assert.equal(isMissingFeeGrantError(missing), true);
    assert.equal(isMissingFeeGrantError({ ...missing, code: 2 }), false);
    assert.equal(
      isMissingFeeGrantError({ ...missing, message: "internal failure" }),
      false,
    );
    assert.equal(
      isMissingFeeGrantError({ ...missing, extra: "unexpected" }),
      false,
    );
  });

  it("normalizes basic, periodic, and nested allowed-message allowances", () => {
    const basic = parseFeeGrantPage(responseWith(basicAllowance())).allowances[0];
    assert.ok(basic);
    assert.equal(basic.supported, true);
    assert.deepEqual(basic.spendLimit, [{ denom: "uzrn", amount: "1000000" }]);
    assert.equal(basic.allowedMessages, null);

    const periodic = parseFeeGrantPage(
      responseWith({
        "@type": PERIODIC_TYPE,
        basic: basicAllowance("9000000"),
        period: "86400s",
        period_spend_limit: [{ denom: "uzrn", amount: "1000000" }],
        period_can_spend: [{ denom: "uzrn", amount: "800000" }],
        period_reset: "2026-07-30T12:00:00Z",
      }),
    ).allowances[0];
    assert.ok(periodic);
    assert.deepEqual(periodic.periodCanSpend, [
      { denom: "uzrn", amount: "800000" },
    ]);
    assert.deepEqual(periodic.periodSpendLimit, [
      { denom: "uzrn", amount: "1000000" },
    ]);
    assert.equal(periodic.periodReset, "2026-07-30T12:00:00Z");

    const restricted = parseFeeGrantPage(
      responseWith({
        "@type": ALLOWED_TYPE,
        allowance: {
          "@type": ALLOWED_TYPE,
          allowance: basicAllowance(),
          allowed_messages: [BANK_SEND_TYPE_URL, CLAIM_TYPE_URL],
        },
        allowed_messages: [BANK_SEND_TYPE_URL],
      }),
    ).allowances[0];
    assert.ok(restricted);
    assert.deepEqual(restricted.allowedMessages, [BANK_SEND_TYPE_URL]);
  });

  it("preserves unknown allowance types without treating them as spendable", () => {
    const grant = parseFeeGrantPage(
      responseWith({ "@type": "/example.future.v1.Allowance" }),
    ).allowances[0];
    assert.ok(grant);
    assert.equal(grant.supported, false);
    assert.equal(
      feeGrantAllowsMessage(grant, BANK_SEND_TYPE_URL, 200_000n, NOW),
      false,
    );
  });

  it("rejects malformed addresses, cursors, duplicates, and excessive nesting", () => {
    assert.throws(
      () =>
        parseFeeGrantPage({
          allowances: [
            {
              granter: `${GRANTER.slice(0, -1)}q`,
              grantee: GRANTEE,
              allowance: basicAllowance(),
            },
          ],
        }),
      /invalid Zerone account address/,
    );
    assert.throws(
      () =>
        parseFeeGrantPage({
          ...responseWith(basicAllowance()) as Record<string, unknown>,
          pagination: { next_key: "***" },
        }),
      /pagination key/,
    );
    assert.throws(
      () =>
        parseFeeGrantPage(
          responseWith({
            ...basicAllowance(),
            spend_limit: [
              { denom: "uzrn", amount: "1" },
              { denom: "uzrn", amount: "2" },
            ],
          }),
        ),
      /malformed coin/,
    );
    assert.throws(
      () =>
        parseFeeGrantPage(
          responseWith(basicAllowance("1", "2026-02-30T12:00:00Z")),
        ),
      /valid UTC timestamp/,
    );
    assert.throws(
      () => parseFeeGrantPage(responseWith(basicAllowance("01"))),
      /malformed coin/,
    );

    let nested: unknown = basicAllowance();
    for (let index = 0; index < 6; index += 1) {
      nested = {
        "@type": ALLOWED_TYPE,
        allowance: nested,
        allowed_messages: [BANK_SEND_TYPE_URL],
      };
    }
    assert.throws(
      () => parseFeeGrantPage(responseWith(nested)),
      /too deep/,
    );
  });
});

describe("feegrant eligibility", () => {
  const grant = parseFeeGrantPage(
    responseWith({
      "@type": ALLOWED_TYPE,
      allowance: basicAllowance("500000", "2026-08-05T12:00:00Z"),
      allowed_messages: [BANK_SEND_TYPE_URL],
    }),
  ).allowances[0]!;

  it("requires support, message scope, a live expiry, and enough native fee cap", () => {
    assert.equal(
      feeGrantAllowsMessage(grant, BANK_SEND_TYPE_URL, 200_000n, NOW),
      true,
    );
    assert.equal(
      feeGrantAllowsMessage(grant, CLAIM_TYPE_URL, 200_000n, NOW),
      false,
    );
    assert.equal(
      feeGrantAllowsMessage(grant, BANK_SEND_TYPE_URL, 600_000n, NOW),
      false,
    );
    assert.equal(
      feeGrantAllowsMessage(
        grant,
        BANK_SEND_TYPE_URL,
        200_000n,
        Date.parse("2026-08-05T12:00:00Z"),
      ),
      false,
    );
  });

  it("uses the next periodic cap when the reset time is due", () => {
    const periodic = parseFeeGrantPage(
      responseWith({
        "@type": PERIODIC_TYPE,
        basic: basicAllowance("9000000"),
        period: "86400s",
        period_spend_limit: [{ denom: "uzrn", amount: "400000" }],
        period_can_spend: [{ denom: "uzrn", amount: "0" }],
        period_reset: "2026-07-29T11:00:00Z",
      }),
    ).allowances[0]!;
    assert.equal(
      feeGrantAllowsMessage(
        periodic,
        BANK_SEND_TYPE_URL,
        200_000n,
        NOW,
      ),
      true,
    );
    assert.equal(
      feeGrantAllowsMessage(
        periodic,
        BANK_SEND_TYPE_URL,
        500_000n,
        NOW,
      ),
      false,
    );
    assert.equal(
      feeGrantAllowsMessage(
        {
          ...periodic,
          periodReset: "2026-07-29T13:00:00Z",
        },
        BANK_SEND_TYPE_URL,
        200_000n,
        NOW,
      ),
      false,
    );
  });
});

describe("bounded feegrant messages", () => {
  it("encodes an allowed-message wrapper around a capped, expiring basic allowance", () => {
    const expiration = new Date(NOW + 7 * 24 * 60 * 60 * 1_000);
    const message = createBoundedGrantMessage({
      granter: GRANTER,
      grantee: GRANTEE,
      spendLimitZrn: "1.25",
      expiration,
      allowedMessages: [BANK_SEND_TYPE_URL, CLAIM_TYPE_URL],
      now: NOW,
    });
    assert.equal(
      message.typeUrl,
      "/cosmos.feegrant.v1beta1.MsgGrantAllowance",
    );
    const grant = message.value as MsgGrantAllowance;
    assert.equal(grant.granter, GRANTER);
    assert.equal(grant.grantee, GRANTEE);
    assert.ok(grant.allowance);
    assert.equal(grant.allowance.typeUrl, ALLOWED_TYPE);
    const restricted = AllowedMsgAllowance.decode(grant.allowance.value);
    assert.deepEqual(restricted.allowedMessages, [
      BANK_SEND_TYPE_URL,
      CLAIM_TYPE_URL,
    ]);
    assert.ok(restricted.allowance);
    assert.equal(restricted.allowance.typeUrl, BASIC_TYPE);
    const basic = BasicAllowance.decode(restricted.allowance.value);
    assert.deepEqual(basic.spendLimit, [
      { denom: "uzrn", amount: "1250000" },
    ]);
    assert.equal(
      basic.expiration?.seconds,
      BigInt(Math.floor(expiration.getTime() / 1_000)),
    );
  });

  it("preserves post-2038 expirations in the encoded allowance", () => {
    const now = Date.parse("2099-01-01T00:00:00Z");
    const expiration = new Date(now + 7 * 24 * 60 * 60 * 1_000 + 123);
    const message = createBoundedGrantMessage({
      granter: GRANTER,
      grantee: GRANTEE,
      spendLimitZrn: "1",
      expiration,
      allowedMessages: [CLAIM_TYPE_URL],
      now,
    });
    const grant = message.value as MsgGrantAllowance;
    const restricted = AllowedMsgAllowance.decode(grant.allowance!.value);
    const basic = BasicAllowance.decode(restricted.allowance!.value);
    assert.deepEqual(basic.expiration, {
      seconds: BigInt(Math.floor(expiration.getTime() / 1_000)),
      nanos: 123_000_000,
    });
  });

  it("rejects self-grants, broad scopes, excessive caps, and excessive duration", () => {
    const base = {
      granter: GRANTER,
      grantee: GRANTEE,
      spendLimitZrn: "1",
      expiration: new Date(NOW + 7 * 24 * 60 * 60 * 1_000),
      allowedMessages: [BANK_SEND_TYPE_URL],
      now: NOW,
    };
    assert.throws(
      () => createBoundedGrantMessage({ ...base, grantee: GRANTER }),
      /different grantee/,
    );
    assert.throws(
      () => createBoundedGrantMessage({ ...base, allowedMessages: [] }),
      /at least one/,
    );
    assert.throws(
      () =>
        createBoundedGrantMessage({
          ...base,
          allowedMessages: ["/cosmos.authz.v1beta1.MsgExec"],
        }),
      /at least one supported/,
    );
    assert.throws(
      () => createBoundedGrantMessage({ ...base, spendLimitZrn: "100.000001" }),
      /capped at 100 ZRN/,
    );
    assert.throws(
      () =>
        createBoundedGrantMessage({
          ...base,
          expiration: new Date(NOW + 31 * 24 * 60 * 60 * 1_000),
        }),
      /30 days/,
    );
  });

  it("encodes an exact granter/grantee revoke", () => {
    const message = createRevokeGrantMessage(GRANTER, GRANTEE);
    assert.equal(
      message.typeUrl,
      "/cosmos.feegrant.v1beta1.MsgRevokeAllowance",
    );
    assert.deepEqual(message.value as MsgRevokeAllowance, {
      granter: GRANTER,
      grantee: GRANTEE,
    });
  });
});
