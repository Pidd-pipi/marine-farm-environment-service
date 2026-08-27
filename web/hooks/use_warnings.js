// useWarnings(filter, pollMs) — warning list hook over GET /api/warnings.
// Shared by the warning desk page and the zone detail page.

import { api } from '/api.js';

export function useWarnings(filter = {}, pollMs = 8000) {
  const store = { data: null, error: null, loading: true, updatedAt: null };
  const subs = new Set();
  let timer = null;
  let stopped = false;

  function buildQuery() {
    const q = new URLSearchParams();
    if (filter.status) q.set('status', filter.status);
    if (filter.zone_id) q.set('zone_id', filter.zone_id);
    if (filter.type) q.set('type', filter.type);
    if (filter.limit) q.set('limit', String(filter.limit));
    const s = q.toString();
    return s ? '/api/warnings?' + s : '/api/warnings';
  }

  function notify() {
    subs.forEach((fn) => {
      try {
        fn(store);
      } catch (e) {
        console.error('[useWarnings] subscriber error', e);
      }
    });
  }

  async function refresh() {
    if (stopped) return;
    try {
      const data = await api(buildQuery());
      store.data = data;
      store.error = null;
      store.updatedAt = new Date();
    } catch (e) {
      store.error = e.message;
    } finally {
      store.loading = false;
      notify();
    }
  }

  return {
    get state() {
      return store;
    },
    subscribe(fn) {
      subs.add(fn);
      fn(store);
      return () => subs.delete(fn);
    },
    start() {
      stopped = false;
      refresh();
      if (pollMs > 0) {
        timer = setInterval(refresh, pollMs);
      }
    },
    stop() {
      stopped = true;
      if (timer) clearInterval(timer);
      timer = null;
    },
    refresh,
  };
}
