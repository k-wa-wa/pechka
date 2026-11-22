import http from "node:http";
import type { AddressInfo } from "node:net";
import { setupServer } from "msw/node";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { createMockServer } from "@/e2e/mock-server.mjs";
import { handlers } from "@/mocks/handlers";

type QueryResult = {
  status: number;
  body: Record<string, unknown>;
};

async function runContractSuite(get: (pathAndQuery: string) => Promise<QueryResult>) {
  // Invalid channel_id -> 400
  const badChannel = await get("/api/v1/admin/contents?channel_id=BAD");
  expect(badChannel.status).toBe(400);

  const badChannelChars = await get("/api/v1/admin/contents?channel_id=-invalid");
  expect(badChannelChars.status).toBe(400);

  // Valid channel_id -> 200
  const okChannel = await get("/api/v1/admin/contents?channel_id=bluray");
  expect(okChannel.status).toBe(200);
  expect(Array.isArray(okChannel.body.contents)).toBe(true);

  // Invalid sort / cursor -> 400
  const badSort = await get("/api/v1/admin/contents?sort=invalid_sort");
  expect(badSort.status).toBe(400);

  const badCursor = await get("/api/v1/admin/contents?after=not-valid-cursor");
  expect(badCursor.status).toBe(400);

  const bothCursors = await get("/api/v1/admin/contents?after=abc&before=def");
  expect(bothCursors.status).toBe(400);

  // Non-integer or out-of-range limit -> falls back to default 20 (matching Go strconv.Atoi)
  for (const invalidLimit of ["1.5", "0", "-5", "101", "abc", "10abc"]) {
    const res = await get(`/api/v1/admin/contents?limit=${encodeURIComponent(invalidLimit)}`);
    expect(res.status).toBe(200);
    expect(res.body.limit).toBe(20);
    expect((res.body.contents as unknown[]).length).toBeLessThanOrEqual(20);
  }

  // Valid integer limit=1 -> respects 1
  const limitOne = await get("/api/v1/admin/contents?limit=1");
  expect(limitOne.status).toBe(200);
  expect(limitOne.body.limit).toBe(1);
  expect((limitOne.body.contents as unknown[]).length).toBe(1);

  // Plus-prefixed integer limit=%2B1 -> matches Go strconv.Atoi("+1") == 1
  const limitPlusOne = await get("/api/v1/admin/contents?limit=%2B1");
  expect(limitPlusOne.status).toBe(200);
  expect(limitPlusOne.body.limit).toBe(1);
  expect((limitPlusOne.body.contents as unknown[]).length).toBe(1);

  // Valid cursor with trailing non-base64url character '!' or standard base64 '+' -> 400
  const validCursor = limitOne.body.next_cursor;
  expect(typeof validCursor).toBe("string");
  const bangCursor = await get(
    `/api/v1/admin/contents?after=${encodeURIComponent(`${String(validCursor)}!`)}`,
  );
  expect(bangCursor.status).toBe(400);
  const plusCursor = await get(
    `/api/v1/admin/contents?after=${encodeURIComponent(`${String(validCursor)}+`)}`,
  );
  expect(plusCursor.status).toBe(400);
}

describe("admin contents API contract parity (MSW & E2E mock-server)", () => {
  describe("MSW handlers", () => {
    const mswServer = setupServer(...handlers);

    beforeAll(() => mswServer.listen({ onUnhandledRequest: "error" }));
    afterAll(() => mswServer.close());

    it("matches backend validation and limit fallback rules", async () => {
      await runContractSuite(async (pathAndQuery) => {
        const res = await fetch(new URL(pathAndQuery, window.location.origin).toString());
        const body = (await res.json()) as Record<string, unknown>;
        return { status: res.status, body };
      });
    });
  });

  describe("E2E mock-server.mjs", () => {
    let server: http.Server;
    let port: number;

    beforeAll(async () => {
      server = createMockServer();
      await new Promise<void>((resolve) => {
        server.listen(0, "127.0.0.1", () => {
          port = (server.address() as AddressInfo).port;
          resolve();
        });
      });
    });

    afterAll(async () => {
      await new Promise<void>((resolve, reject) => {
        server.close((err) => (err ? reject(err) : resolve()));
      });
    });

    it("matches backend validation and limit fallback rules", async () => {
      await runContractSuite(
        (pathAndQuery) =>
          new Promise<QueryResult>((resolve, reject) => {
            http
              .get(`http://127.0.0.1:${port}${pathAndQuery}`, (res) => {
                let raw = "";
                res.on("data", (chunk) => {
                  raw += chunk;
                });
                res.on("end", () => {
                  resolve({
                    status: res.statusCode ?? 0,
                    body: JSON.parse(raw) as Record<string, unknown>,
                  });
                });
              })
              .on("error", reject);
          }),
      );
    });
  });
});
