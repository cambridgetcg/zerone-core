import assert from "node:assert/strict";
import test from "node:test";

import { Miniflare } from "miniflare";

type D1Meta = {
  changes: number;
};

type ObjectRowsResult = {
  success: boolean;
  results: Array<Record<string, unknown>>;
  meta: D1Meta;
};

type ColumnRowsResult = {
  success: boolean;
  results: {
    columns: string[];
    rows: unknown[][];
  };
  meta: D1Meta;
};

type D1Statement = {
  sql: string;
  params: unknown[];
};

const D1_PROXY_SCRIPT = `
  addEventListener("fetch", (event) => {
    event.respondWith(__D1_BETA__DB.fetch(event.request));
  });
`;

async function queryD1<T>(
  miniflare: Miniflare,
  statements: D1Statement | D1Statement[],
  resultsFormat: "ARRAY_OF_OBJECTS" | "ROWS_AND_COLUMNS",
): Promise<T[]> {
  const response = await miniflare.dispatchFetch(
    `http://d1/query?resultsFormat=${resultsFormat}`,
    {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(statements),
    },
  );

  assert.equal(response.status, 200);
  return (await response.json()) as T[];
}

function objectRows(result: ColumnRowsResult): Array<Record<string, unknown>> {
  return result.results.rows.map((row) =>
    Object.fromEntries(
      result.results.columns.map((column, index) => [column, row[index]]),
    ),
  );
}

test("Workerd D1 keeps RETURNING rows distinct from trigger-inclusive changes", async () => {
  // The legacy/raw binding bypasses only Workerd's JS D1 facade. It reaches the
  // same Miniflare D1 durable object and /query transaction endpoint used by
  // D1Database.batch(), so this remains an actual Workerd SQLite parity test.
  const miniflare = new Miniflare({
    script: D1_PROXY_SCRIPT,
    compatibilityDate: "2026-07-22",
    d1Databases: { __D1_BETA__DB: "d1-returning-parity" },
    d1Persist: false,
  });

  try {
    const schema = await queryD1<ObjectRowsResult>(
      miniflare,
      [
        {
          sql: "CREATE TABLE parent (id INTEGER PRIMARY KEY, value TEXT NOT NULL UNIQUE)",
          params: [],
        },
        {
          sql: "CREATE TABLE audit (id INTEGER PRIMARY KEY, parent_id INTEGER NOT NULL)",
          params: [],
        },
        {
          sql: "CREATE TRIGGER parent_after_insert AFTER INSERT ON parent BEGIN INSERT INTO audit (parent_id) VALUES (NEW.id); END",
          params: [],
        },
      ],
      "ARRAY_OF_OBJECTS",
    );
    assert.equal(schema.length, 3);
    assert.ok(schema.every((result) => result.success));

    const [triggeredInsert] = await queryD1<ObjectRowsResult>(
      miniflare,
      {
        sql: "INSERT INTO parent (value) VALUES (?) RETURNING id, value",
        params: ["alpha"],
      },
      "ARRAY_OF_OBJECTS",
    );

    assert.ok(triggeredInsert);
    assert.deepEqual(triggeredInsert.results, [{ id: 1, value: "alpha" }]);
    assert.equal(triggeredInsert.meta.changes, 2);
    assert.ok(triggeredInsert.meta.changes > triggeredInsert.results.length);

    const [auditCount] = await queryD1<ObjectRowsResult>(
      miniflare,
      { sql: "SELECT COUNT(*) AS count FROM audit", params: [] },
      "ARRAY_OF_OBJECTS",
    );
    assert.ok(auditCount);
    assert.deepEqual(auditCount.results, [{ count: 1 }]);

    // D1Database.batch() sends its statements to this endpoint in
    // ROWS_AND_COLUMNS mode and maps the returned columns/rows to objects.
    const batch = await queryD1<ColumnRowsResult>(
      miniflare,
      [
        {
          sql: "INSERT INTO parent (value) VALUES (?) RETURNING id, value",
          params: ["beta"],
        },
        {
          sql: "INSERT INTO parent (value) SELECT ? WHERE NOT EXISTS (SELECT 1 FROM parent WHERE value = ?) RETURNING id, value",
          params: ["alpha", "alpha"],
        },
      ],
      "ROWS_AND_COLUMNS",
    );

    assert.equal(batch.length, 2);
    assert.ok(batch.every((result) => result.success));
    assert.ok(batch[0]);
    assert.ok(batch[1]);
    assert.deepEqual(objectRows(batch[0]), [{ id: 2, value: "beta" }]);
    assert.equal(batch[0].meta.changes, 2);
    assert.deepEqual(objectRows(batch[1]), []);
    assert.equal(batch[1].meta.changes, 0);
  } finally {
    await miniflare.dispose();
  }
});
