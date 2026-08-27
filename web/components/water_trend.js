// WaterTrend — reusable water-quality trend chart (SVG).
// Shared by: zone detail page (full trend) and warning desk page (trend
// preview of the selected zone). Renders dissolved oxygen as a line with
// the zone warning/danger thresholds as guide bands.

import { fmtNum, fmtTime, escapeHtml } from '/api.js';

const W = 720;
const H = 240;
const PAD = { top: 18, right: 20, bottom: 28, left: 44 };

export function WaterTrend(samples, { warnThreshold = 4, dangerThreshold = 3, title = '溶解氧趋势 (mg/L)' } = {}) {
  const panel = document.createElement('div');
  panel.className = 'trend-panel';
  const list = (samples || []).slice().sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
  const empty = list.length === 0;
  if (empty) {
    panel.innerHTML = '<div class="empty-state">暂无水质采样数据</div>';
    return panel;
  }

  const maxDO = Math.max(12, ...list.map((s) => Number(s.do) || 0));
  const yMax = Math.min(16, Math.ceil(maxDO * 1.15));
  const yMin = 0;
  const xMin = new Date(list[0].timestamp).getTime();
  const xMax = new Date(list[list.length - 1].timestamp).getTime();
  const xSpan = Math.max(xMax - xMin, 1);

  const sx = (t) => PAD.left + ((new Date(t).getTime() - xMin) / xSpan) * (W - PAD.left - PAD.right);
  const sy = (v) => PAD.top + ((yMax - v) / (yMax - yMin)) * (H - PAD.top - PAD.bottom);

  const pts = list.map((s) => `${sx(s.timestamp).toFixed(1)},${sy(Number(s.do) || 0).toFixed(1)}`).join(' ');
  const linePath = list.map((s, i) => `${i === 0 ? 'M' : 'L'}${sx(s.timestamp).toFixed(1)} ${sy(Number(s.do) || 0).toFixed(1)}`).join(' ');

  const grid = [];
  for (let v = 0; v <= yMax; v += 2) {
    grid.push(`<line x1="${PAD.left}" y1="${sy(v)}" x2="${W - PAD.right}" y2="${sy(v)}" stroke="#e2e8f0" stroke-width="1"/>`);
    grid.push(`<text x="${PAD.left - 8}" y="${sy(v) + 4}" text-anchor="end" font-size="10" fill="#94a3b8">${v}</text>`);
  }

  const xTicks = 6;
  const tickStep = Math.max(1, Math.floor(list.length / xTicks));
  const xLabels = list
    .filter((_, i) => i % tickStep === 0)
    .map((s) => `<text x="${sx(s.timestamp)}" y="${H - 8}" text-anchor="middle" font-size="10" fill="#94a3b8">${fmtTime(s.timestamp).slice(5, 16)}</text>`)
    .join('');

  const latest = list[list.length - 1];
  panel.innerHTML = `
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:6px">
      <strong>${escapeHtml(title)}</strong>
      <span class="muted">最新 ${fmtNum(latest.do, 2)} mg/L · ${fmtTime(latest.timestamp)}</span>
    </div>
    <svg viewBox="0 0 ${W} ${H}" role="img" aria-label="溶解氧趋势">
      ${grid.join('')}
      <rect x="${PAD.left}" y="${sy(dangerThreshold)}" width="${W - PAD.left - PAD.right}" height="${sy(0) - sy(dangerThreshold)}" fill="#fee2e2" opacity="0.35"/>
      <rect x="${PAD.left}" y="${sy(warnThreshold)}" width="${W - PAD.left - PAD.right}" height="${sy(dangerThreshold) - sy(warnThreshold)}" fill="#fef3c7" opacity="0.4"/>
      <line x1="${PAD.left}" y1="${sy(dangerThreshold)}" x2="${W - PAD.right}" y2="${sy(dangerThreshold)}" stroke="#dc2626" stroke-width="1" stroke-dasharray="4 3"/>
      <text x="${W - PAD.right - 2}" y="${sy(dangerThreshold) - 4}" text-anchor="end" font-size="10" fill="#dc2626">危险 ${dangerThreshold}</text>
      <line x1="${PAD.left}" y1="${sy(warnThreshold)}" x2="${W - PAD.right}" y2="${sy(warnThreshold)}" stroke="#b45309" stroke-width="1" stroke-dasharray="4 3"/>
      <text x="${W - PAD.right - 2}" y="${sy(warnThreshold) - 4}" text-anchor="end" font-size="10" fill="#b45309">预警 ${warnThreshold}</text>
      <polyline points="${pts}" fill="none" stroke="#0e7490" stroke-width="2"/>
      ${list.map((s) => {
        const v = Number(s.do) || 0;
        const color = v < dangerThreshold ? '#dc2626' : v < warnThreshold ? '#b45309' : '#0e7490';
        return `<circle cx="${sx(s.timestamp)}" cy="${sy(v)}" r="2.6" fill="${color}"/>`;
      }).join('')}
      ${xLabels}
    </svg>`;
  return panel;
}
