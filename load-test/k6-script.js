/**
 * pechka API 負荷テスト
 *
 * 使い方:
 *   k6 run --env BASE_URL=http://localhost:8080 load-test/k6-script.js
 *
 * オプション:
 *   --env BASE_URL=<API URL>   デフォルト: http://localhost:8080
 *   --env VUS=<仮想ユーザ数>   デフォルト: 10
 *   --env DURATION=<秒>        デフォルト: 30s
 */

import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

const errorRate = new Rate("error_rate");
const listLatency = new Trend("list_contents_latency");
const detailLatency = new Trend("content_detail_latency");
const searchLatency = new Trend("search_latency");

export const options = {
  scenarios: {
    // コンテンツ一覧の連続アクセス（最も頻度が高い操作）
    list_contents: {
      executor: "constant-vus",
      vus: parseInt(__ENV.VUS || "10"),
      duration: __ENV.DURATION || "30s",
      exec: "listContents",
    },
    // 検索クエリ（やや重い操作）
    search: {
      executor: "constant-vus",
      vus: 3,
      duration: __ENV.DURATION || "30s",
      exec: "searchContents",
      startTime: "5s",
    },
  },
  thresholds: {
    // 95パーセンタイルが500ms以内
    http_req_duration: ["p(95)<500"],
    // エラー率1%以内
    error_rate: ["rate<0.01"],
    list_contents_latency: ["p(95)<300"],
    search_latency: ["p(95)<800"],
  },
};

export function listContents() {
  const res = http.get(`${BASE_URL}/v1/contents`, {
    headers: { Accept: "application/json" },
  });

  const ok = check(res, {
    "list: status 200": (r) => r.status === 200,
    "list: has body": (r) => r.body && r.body.length > 0,
  });

  errorRate.add(!ok);
  listLatency.add(res.timings.duration);
  sleep(0.5);
}

export function searchContents() {
  const queries = ["test", "video", "movie", "action"];
  const q = queries[Math.floor(Math.random() * queries.length)];

  const res = http.get(`${BASE_URL}/v1/search?q=${q}`, {
    headers: { Accept: "application/json" },
  });

  const ok = check(res, {
    "search: status 200": (r) => r.status === 200,
  });

  errorRate.add(!ok);
  searchLatency.add(res.timings.duration);
  sleep(1);
}

export function contentDetail() {
  // short_id が必要なため、先に一覧を取得してIDを使う
  const listRes = http.get(`${BASE_URL}/v1/contents?limit=1`);
  if (listRes.status !== 200) {
    errorRate.add(1);
    return;
  }

  let contents;
  try {
    contents = JSON.parse(listRes.body);
  } catch {
    errorRate.add(1);
    return;
  }

  if (!contents || contents.length === 0) {
    sleep(1);
    return;
  }

  const id = contents[0].short_id;
  const res = http.get(`${BASE_URL}/v1/contents/${id}`, {
    headers: { Accept: "application/json" },
  });

  const ok = check(res, {
    "detail: status 200": (r) => r.status === 200,
  });

  errorRate.add(!ok);
  detailLatency.add(res.timings.duration);
  sleep(0.5);
}

export function handleSummary(data) {
  return {
    stdout: summaryText(data),
  };
}

function summaryText(data) {
  const metrics = data.metrics;
  const lines = [
    "=== pechka 負荷テスト結果 ===",
    `実行時間: ${new Date().toISOString()}`,
    "",
    "--- HTTP リクエスト ---",
    `総リクエスト数: ${metrics.http_reqs?.values?.count || 0}`,
    `エラー率: ${((metrics.error_rate?.values?.rate || 0) * 100).toFixed(2)}%`,
    "",
    "--- レイテンシ (http_req_duration) ---",
    `p50: ${(metrics.http_req_duration?.values?.["p(50)"] || 0).toFixed(1)}ms`,
    `p90: ${(metrics.http_req_duration?.values?.["p(90)"] || 0).toFixed(1)}ms`,
    `p95: ${(metrics.http_req_duration?.values?.["p(95)"] || 0).toFixed(1)}ms`,
    `p99: ${(metrics.http_req_duration?.values?.["p(99)"] || 0).toFixed(1)}ms`,
    "",
    "--- エンドポイント別 ---",
    `コンテンツ一覧 p95: ${(metrics.list_contents_latency?.values?.["p(95)"] || 0).toFixed(1)}ms`,
    `検索 p95:          ${(metrics.search_latency?.values?.["p(95)"] || 0).toFixed(1)}ms`,
  ];
  return lines.join("\n") + "\n";
}
