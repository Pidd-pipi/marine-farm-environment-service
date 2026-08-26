// ZoneCard — reusable farm-zone status card.
// Shared by: overview page (zone grid) and zone detail page (current
// status header). Consumes the per-zone aggregation from /api/overview.

import { fmtNum, fmtTime, escapeHtml } from '/api.js';
import { ZONE_STATUS_LABEL, ZONE_STATUS_CLASS, AERATOR_STATUS_LABEL, AERATOR_STATUS_CLASS } from '/enums.js';

export function ZoneCard(zo, { onSelect, compact } = {}) {
  const zone = zo.zone || {};
  const status = zone.status || 'normal';
  const latest = zo.latest_sample || null;
  const aerStatus = zo.aerator_status || '';
  const card = document.createElement('article');
  card.className = 'zone-card';
  card.tabIndex = 0;

  const aerHtml = aerStatus
    ? `<span class="status-badge ${AERATOR_STATUS_CLASS[aerStatus] || 'status-stopped'}">增氧 · ${AERATOR_STATUS_LABEL[aerStatus] || aerStatus}</span>`
    : '<span class="muted">增氧机未运行</span>';

  card.innerHTML = `
    <div class="zone-card-head">
      <div>
        <h3 class="zone-name">${escapeHtml(zone.name || zone.id || '-')}</h3>
        <div class="zone-sub">${escapeHtml(zone.id || '')} · 面积 ${fmtNum(zone.area, 0)} 亩 · 存塘 ${fmtInt(zone.stock)}</div>
      </div>
      <span class="status-badge ${ZONE_STATUS_CLASS[status] || ''}">${ZONE_STATUS_LABEL[status] || status}</span>
    </div>
    ${zone.restore_eligible ? '<div><span class="restore-flag">✓ 可确认恢复</span></div>' : ''}
    <div class="zone-card-body">
      <div class="metrics">
        <div class="metric">
          <span class="metric-label">溶解氧</span>
          <span class="metric-value ${latest && latest.do < 4 ? 'metric-alert' : ''}">${latest ? fmtNum(latest.do, 2) : '-'}</span>
        </div>
        <div class="metric"><span class="metric-label">水温</span><span class="metric-value">${latest ? fmtNum(latest.temperature, 1) : '-'}</span></div>
        <div class="metric"><span class="metric-label ${zo.open_warning_count ? 'metric-alert' : ''}">未处置预警</span><span class="metric-value">${zo.open_warning_count || 0}</span></div>
      </div>
      <div class="latest-sample">
        <span>浮标 <b>${zo.buoy_count || 0}</b></span>
        <span>离线 <b>${zo.stale_buoy_count || 0}</b></span>
        <span>pH <b>${latest ? fmtNum(latest.ph, 2) : '-'}</b></span>
        <span>盐度 <b>${latest ? fmtNum(latest.salinity, 1) : '-'}</b></span>
        <span>氨氮 <b>${latest ? fmtNum(latest.ammonia, 2) : '-'}</b></span>
        <span>上报 <b>${latest ? fmtTime(latest.timestamp) : '暂无'}</b></span>
      </div>
      <div class="zone-card-foot">
        <span class="muted">状态自 ${fmtTime(zone.status_since)}</span>
        ${aerHtml}
      </div>
    </div>`;

  card.addEventListener('click', () => {
    if (onSelect) onSelect(zone.id);
    else window.location.href = '/zones/' + encodeURIComponent(zone.id);
  });
  card.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') card.click();
  });
  if (compact) card.classList.add('zone-card-compact');
  return card;
}

export function fmtInt(v) {
  if (v === null || v === undefined) return '-';
  return Number(v).toLocaleString('en-US');
}
