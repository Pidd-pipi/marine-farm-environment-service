// api.js — unified fetch wrapper for the backend REST API.
// Every endpoint returns the envelope {code, message, data}. This module
// unwraps `data` and throws a descriptive Error on failure.

export async function api(path, options = {}) {
  const headers = Object.assign(
    { 'Content-Type': 'application/json', 'X-Request-Id': newRequestId() },
    options.headers || {},
  );
  const res = await fetch(path, Object.assign({}, options, { headers }));
  let body = null;
  try {
    body = await res.json();
  } catch (e) {
    // non-JSON response
  }
  if (!res.ok || !body || body.code !== 0) {
    const err = new Error((body && body.message) || `HTTP ${res.status}`);
    err.status = res.status;
    err.code = body && body.error;
    err.requestId = body && body.request_id;
    throw err;
  }
  return body.data;
}

export async function post(path, payload) {
  return api(path, { method: 'POST', body: JSON.stringify(payload || {}) });
}

export function newRequestId() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return 'web-' + crypto.randomUUID().slice(0, 8);
  }
  return 'web-' + Date.now().toString(36);
}

export function fmtTime(iso) {
  if (!iso) return '-';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString('zh-CN', { hour12: false });
}

export function fmtDate(iso) {
  if (!iso) return '-';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString('zh-CN');
}

export function fmtNum(v, digits = 1) {
  if (v === null || v === undefined || Number.isNaN(Number(v))) return '-';
  return Number(v).toLocaleString('en-US', { maximumFractionDigits: digits });
}

export function fmtInt(v) {
  if (v === null || v === undefined) return '-';
  return Number(v).toLocaleString('en-US');
}

export function escapeHtml(s) {
  if (s === null || s === undefined) return '';
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}
