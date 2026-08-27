// WarningTable — reusable warning table with status badges and optional
// row actions (verify / resolve). Shared by: warning desk page and overview
// page (recent warnings).

import { fmtTime, fmtNum, escapeHtml } from '/api.js';
import {
  WARNING_TYPE_LABEL, WARNING_STATUS_LABEL, WARNING_STATUS_CLASS,
  WARNING_LEVEL_LABEL,
} from '/enums.js';

export function WarningTable(warnings, { showActions = true, zoneNameOf, onChanged } = {}) {
  const wrap = document.createElement('div');
  wrap.className = 'table-wrap';
  const list = warnings || [];

  if (!list.length) {
    wrap.innerHTML = '<div class="empty-state">暂无预警记录 ✓</div>';
    return wrap;
  }

  const rows = list.map((w) => {
    const zoneName = zoneNameOf ? zoneNameOf(w.zone_id) : w.zone_id;
    return `
      <tr data-warning-id="${escapeHtml(w.id)}" data-zone-id="${escapeHtml(w.zone_id)}">
        <td><span class="status-badge ${WARNING_STATUS_CLASS[w.status] || ''}">${WARNING_STATUS_LABEL[w.status] || w.status}</span></td>
        <td><span class="status-badge status-danger">${escapeHtml(w.level || '-')}</span></td>
        <td><b>${WARNING_TYPE_LABEL[w.type] || escapeHtml(w.type)}</b></td>
        <td>${escapeHtml(zoneName || w.zone_id || '-')}</td>
        <td>${escapeHtml(w.buoy_id || '-')}</td>
        <td class="num">${fmtNum(w.do, 2)}</td>
        <td class="num">${fmtNum(w.temperature, 1)}</td>
        <td class="num">${fmtNum(w.ph, 2)}</td>
        <td class="num">${fmtNum(w.ammonia, 2)}</td>
        <td>${w.cross_checked ? (w.cross_check_ok ? '<span class="status-badge status-pending">交叉验证存疑</span>' : '<span class="status-badge status-resolved">交叉验证通过</span>') : ''}</td>
        <td>${fmtTime(w.reported_at)}</td>
        <td title="${escapeHtml(w.detail)}">${escapeHtml((w.detail || '').slice(0, 24))}</td>
        <td>
          ${showActions && w.status === 'pending' ? `<button class="btn btn-sm btn-primary act-verify">核实</button> ` : ''}
          ${showActions && w.status !== 'resolved' ? `<button class="btn btn-sm act-resolve">解除</button>` : ''}
        </td>
      </tr>`;
  }).join('');

  wrap.innerHTML = `
    <table>
      <thead>
        <tr>
          <th>状态</th><th>级别</th><th>类型</th><th>养殖区</th><th>浮标</th>
          <th class="num">溶解氧</th><th class="num">水温</th><th class="num">pH</th><th class="num">氨氮</th>
          <th>交叉验证</th><th>上报时间</th><th>说明</th><th>操作</th>
        </tr>
      </thead>
      <tbody>${rows}</tbody>
    </table>`;

  wrap.querySelectorAll('.act-verify').forEach((btn) => {
    btn.addEventListener('click', async (e) => {
      e.stopPropagation();
      const id = btn.closest('tr').dataset.warningId;
      btn.disabled = true;
      try {
        await apiPost(`/api/warnings/${encodeURIComponent(id)}/verify`);
        if (onChanged) onChanged('verified', id);
      } catch (err) {
        window.alert('核实失败：' + err.message);
        btn.disabled = false;
      }
    });
  });

  wrap.querySelectorAll('.act-resolve').forEach((btn) => {
    btn.addEventListener('click', async (e) => {
      e.stopPropagation();
      const id = btn.closest('tr').dataset.warningId;
      btn.disabled = true;
      try {
        await apiPost(`/api/warnings/${encodeURIComponent(id)}/resolve`);
        if (onChanged) onChanged('resolved', id);
      } catch (err) {
        window.alert('解除失败：' + err.message);
        btn.disabled = false;
      }
    });
  });

  return wrap;
}

// Local POST helper keeps this component dependency-light.
async function apiPost(path) {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  });
  const body = await res.json().catch(() => null);
  if (!res.ok || !body || body.code !== 0) {
    throw new Error((body && body.message) || `HTTP ${res.status}`);
  }
  return body.data;
}
