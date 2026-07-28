import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import { MsgRegisterAccount, MsgRotateKey } from "../src/generated/zerone/auth/v1/tx";
import { MsgCreatePool } from "../src/generated/zerone/liquiditypool/v1/tx";
import { FactCitation, PendingClaim, SubstrateLink } from "../src/generated/zerone/substrate_bridge/v1/substrate_link";
import { MsgSubmitExternalAttestation } from "../src/generated/zerone/substrate_bridge/v1/tx";
import {
  AxisProjection,
  CitationType,
  ExternalSource,
} from "../src/generated/zerone/substrate_bridge/v1/types";

interface WireFixture {
  schema: string;
  vectors: WireVector[];
}

interface WireVector {
  name: string;
  typeUrl: string;
  value: unknown;
  hex: string;
}

interface WireCodec<T> {
  readonly typeUrl: string;
  encode(message: T): { finish(): Uint8Array };
  decode(input: Uint8Array): T;
}

const fixture = JSON.parse(
  readFileSync(new URL("../testdata/wire-vectors.json", import.meta.url), "utf8"),
) as WireFixture;

describe("Go and TypeScript protobuf wire parity", () => {
  assert.equal(fixture.schema, "zerone-typescript-sdk-wire-vectors/v1");

  for (const vector of fixture.vectors) {
    it(vector.name, () => {
      const golden = Uint8Array.from(Buffer.from(vector.hex, "hex"));
      const { codec, encoded } = encodeVector(vector);

      assert.equal(Buffer.from(encoded).toString("hex"), vector.hex);
      assert.deepEqual(codec.encode(codec.decode(golden)).finish(), golden);
    });
  }
});

function encodeVector(vector: WireVector): {
  codec: WireCodec<never>;
  encoded: Uint8Array;
} {
  const value = asObject(vector.value, "value");

  switch (vector.typeUrl) {
    case MsgRegisterAccount.typeUrl: {
      const message = MsgRegisterAccount.fromPartial({
        sender: asString(value.sender, "sender"),
        did: asString(value.did, "did"),
        publicKey: asString(value.publicKey, "publicKey"),
        accountType: asString(value.accountType, "accountType"),
        operationalKeyHash: asString(
          value.operationalKeyHash,
          "operationalKeyHash",
        ),
        metadata: asString(value.metadata, "metadata"),
      });
      return wireResult(MsgRegisterAccount, message);
    }
    case MsgRotateKey.typeUrl: {
      const message = MsgRotateKey.fromPartial({
        sender: asString(value.sender, "sender"),
        newOperationalKey: asBytes(value.newOperationalKey, "newOperationalKey"),
        authorizationSignature: asBytes(
          value.authorizationSignature,
          "authorizationSignature",
        ),
      });
      return wireResult(MsgRotateKey, message);
    }
    case MsgCreatePool.typeUrl: {
      const message = MsgCreatePool.fromPartial({
        creator: asString(value.creator, "creator"),
        denomA: asString(value.denomA, "denomA"),
        denomB: asString(value.denomB, "denomB"),
        amountA: asString(value.amountA, "amountA"),
        amountB: asString(value.amountB, "amountB"),
        swapFeeBps: asBigInt(value.swapFeeBps, "swapFeeBps"),
      });
      return wireResult(MsgCreatePool, message);
    }
    case MsgSubmitExternalAttestation.typeUrl: {
      const message = MsgSubmitExternalAttestation.fromPartial({
        submitter: asString(value.submitter, "submitter"),
        adapterId: asString(value.adapterId, "adapterId"),
        workClassId: asString(value.workClassId, "workClassId"),
        link: parseSubstrateLink(value.link),
        bondUzrn: asString(value.bondUzrn, "bondUzrn"),
      });
      return wireResult(MsgSubmitExternalAttestation, message);
    }
    default:
      throw new Error(`unsupported TypeScript SDK wire-vector ${vector.typeUrl}`);
  }
}

function wireResult<T>(
  codec: WireCodec<T>,
  message: T,
): { codec: WireCodec<never>; encoded: Uint8Array } {
  return {
    codec: codec as WireCodec<never>,
    encoded: codec.encode(message).finish(),
  };
}

function parseSubstrateLink(input: unknown): SubstrateLink {
  const value = asObject(input, "link");
  const citedFacts = asArray(value.citedFacts, "citedFacts").map((entry) => {
    const citation = asObject(entry, "citedFact");
    return FactCitation.fromPartial({
      factId: asString(citation.factId, "factId"),
      citationType: asCitationType(citation.citationType),
      citationContext: asString(citation.citationContext, "citationContext"),
    });
  });
  const pendingClaims = asArray(value.pendingClaims, "pendingClaims").map(
    (entry) => {
      const claim = asObject(entry, "pendingClaim");
      return PendingClaim.fromPartial({
        claimContent: asString(claim.claimContent, "claimContent"),
        proposedFactId: asString(claim.proposedFactId, "proposedFactId"),
        domain: asString(claim.domain, "domain"),
        methodologyId: asString(claim.methodologyId, "methodologyId"),
        relations: asArray(claim.relations, "relations").map((relationEntry) => {
          const relation = asObject(relationEntry, "relation");
          return {
            targetFactId: asString(relation.targetFactId, "targetFactId"),
            relation: asString(relation.relation, "relation"),
            inference: asString(relation.inference, "inference"),
            inferenceStrengthBps: asNumber(
              relation.inferenceStrengthBps,
              "inferenceStrengthBps",
            ),
          };
        }),
      });
    },
  );
  const recursionWeight = asObject(value.recursionWeight, "recursionWeight");
  const source = asObject(value.source, "source");

  return SubstrateLink.fromPartial({
    citedFacts,
    pendingClaims,
    recursionWeight: AxisProjection.fromPartial({
      axisSubstrate: asBigInt(
        recursionWeight.axisSubstrate,
        "axisSubstrate",
      ),
      axisVerification: asBigInt(
        recursionWeight.axisVerification,
        "axisVerification",
      ),
      axisClassification: asBigInt(
        recursionWeight.axisClassification,
        "axisClassification",
      ),
      axisAttribution: asBigInt(
        recursionWeight.axisAttribution,
        "axisAttribution",
      ),
      axisTooling: asBigInt(recursionWeight.axisTooling, "axisTooling"),
      axisInterface: asBigInt(recursionWeight.axisInterface, "axisInterface"),
    }),
    adapterId: asString(value.adapterId, "link.adapterId"),
    source: ExternalSource.fromPartial({
      adapterId: asString(source.adapterId, "source.adapterId"),
      sourceId: asString(source.sourceId, "source.sourceId"),
      sourceUrl: asString(source.sourceUrl, "source.sourceUrl"),
      contentHash: asBytes(source.contentHash, "source.contentHash"),
      fetchedAtBlock: asBigInt(source.fetchedAtBlock, "source.fetchedAtBlock"),
    }),
    linkHash: asBytes(value.linkHash, "linkHash"),
  });
}

function asObject(
  input: unknown,
  label: string,
): Record<string, unknown> {
  assert.ok(
    input !== null && typeof input === "object" && !Array.isArray(input),
    `${label} must be an object`,
  );
  return input as Record<string, unknown>;
}

function asArray(input: unknown, label: string): unknown[] {
  assert.ok(Array.isArray(input), `${label} must be an array`);
  return input;
}

function asString(input: unknown, label: string): string {
  if (typeof input !== "string") {
    throw new TypeError(`${label} must be a string`);
  }
  return input;
}

function asBytes(input: unknown, label: string): Uint8Array {
  return Uint8Array.from(Buffer.from(asString(input, label), "base64"));
}

function asBigInt(input: unknown, label: string): bigint {
  return BigInt(asString(input, label));
}

function asNumber(input: unknown, label: string): number {
  if (typeof input !== "number") {
    throw new TypeError(`${label} must be a number`);
  }
  return input;
}

function asCitationType(input: unknown): CitationType {
  const name = asString(input, "citationType");
  const citationType = CitationType[name as keyof typeof CitationType];
  assert.equal(typeof citationType, "number", `unknown citation type ${name}`);
  return citationType;
}
