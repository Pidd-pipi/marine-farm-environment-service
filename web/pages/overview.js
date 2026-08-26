// overview.js — 牧场总览 dashboard (GET /). Uses ZoneCard + WarningTable +
// useZones. Consumes GET /api/overview.

import { useZones } from '/hooks/use_zones.js';
import { ZoneCard } from '/components/zone_card.js';
import { WarningTable } from '/components/warning_table.js';
import { fmtTime, escapeHtml } from '/api.js';

export function render(container) {
  const zones = useZones(4000);
  zones.start();
  const unsubZones = zones.subscribe((s) => renderOverview(container, s));

  return () => {
    unsubZones();
    zones.stop();
  };
}

function renderOverview(container, state) {
  if (state.loading && !state.data) {
    container.innerHTML = '<div class="page-loading">加载总览…</div>';
    return;
  }
  if (state.error && !state.data) {
    container.innerHTML = `<div class="error-state">加载失败：${escapeHtml(state.error)}</div>`;
    return;
  }
  const data = state.data || { zones: [], totals: {}, recent_warnings: [] };
  const zones = data.zones || [];
  const totals = data.totals || {};

  container.innerHTML = `
    <div class="page-head">
      <div>
        <h2>牧场总览</h2>
        <p class="muted">养殖区状态 · 溶解氧实时值 · 预警计数（${fmtTime(state.updatedAt ? state.updatedAt.toISOString() : null)} 更新）</p>
      </div>
      <div class="stat-strip">
        <div class="stat"><span class="stat-num">${totals.zone_count ?? zones.length}</span><span class="stat-label">养殖区</span></div>
        <div class="stat"><span class="stat-num">${totals.buoy_count ?? 0}</span><span class="stat-label">监测浮标</span></div>
        <div class="stat stat-alert"><span class="stat-num">${totals.open_warning_count ?? 0}</span><span class="stat-label">未处置预警</span></div>
        <div class="stat stat-aerate"><span class="stat-num">${totals.active_aerators ?? 0}</span><span class="stat-label">运行增氧机</span></div>
      </div>
    </div>
    <section>
      <h3 class="section-title">养殖区卡片</h3>
      <div class="zone-grid" id="zone-grid"></div>
    </section>
    <section>
      <h3 class="section-title">最新预警</h3>
      <div id="recent-warnings"></div>
    </section>`;

  const grid = container.querySelector('#zone-grid');
  for (const z of zones) {
    grid.appendChild(ZoneCard(z));
  }
  if (!zones.length) {
    grid.innerHTML = '<div class="empty-state">暂无养殖区</div>';
  }

  const recent = container.querySelector('#recent-warnings');
  const zoneNameOf = (id) => {
    const z = zones.find((x) => x.zone && x.zone.id === id);
    return z && z.zone.name;
  };
  recent.appendChild(WarningTable(data.recent_warnings || [], {
    showActions: false,
    zoneNameOf,
  }));
}
