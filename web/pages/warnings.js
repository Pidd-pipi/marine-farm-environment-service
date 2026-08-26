// warnings.js — 预警台 (/warnings). Uses WarningTable + WaterTrend +
// useWarnings. Filter by status/type and preview the DO trend of a zone.

import { useWarnings } from '/hooks/use_warnings.js';
import { useZones } from '/hooks/use_zones.js';
import { WarningTable } from '/components/warning_table.js';
import { WaterTrend } from '/components/water_trend.js';
import { api, fmtTime, escapeHtml } from '/api.js';
import { WARNING_TYPE_LABEL } from '/enums.js';

export function render(container) {
  let filter = { status: '', type: '', limit: 100 };
  const warnings = useWarnings(filter, 6000);
  const zones = useZones(15000);
  warnings.start();
  zones.start();

  let samplesCache = [];

  const unsubWarnings = warnings.subscribe(() => renderWarnings(container, warnings.state, zones.state));
  const unsubZones = zones.subscribe((s) => {
    if (s.data) renderWarnings(container, warnings.state, s);
  });

  function renderWarnings(c, wstate, zstate) {
    if (wstate.loading && !wstate.data) {
      c.innerHTML = '<div class="page-loading">加载预警台…</div>';
      return;
    }
    const list = wstate.data || [];
    const zoneMap = {};
    (zstate.data ? zstate.data.zones || [] : []).forEach((z) => {
      if (z.zone) zoneMap[z.zone.id] = z.zone;
    });

    c.innerHTML = `
      <div class="page-head">
        <div>
          <h2>预警台</h2>
          <p class="muted">预警核实与处置（${fmtTime(wstate.updatedAt ? wstate.updatedAt.toISOString() : null)} 更新）</p>
        </div>
        <div class="form-row">
          <div class="field">
            <label>状态</label>
            <select id="f-status">
              <option value="">全部</option>
              <option value="pending" ${filter.status === 'pending' ? 'selected' : ''}>待核实</option>
              <option value="confirmed" ${filter.status === 'confirmed' ? 'selected' : ''}>已确认</option>
              <option value="resolved" ${filter.status === 'resolved' ? 'selected' : ''}>已解除</option>
            </select>
          </div>
          <div class="field">
            <label>类型</label>
            <select id="f-type">
              <option value="">全部</option>
              ${Object.entries(WARNING_TYPE_LABEL).map(([k, v]) =>
                `<option value="${k}" ${filter.type === k ? 'selected' : ''}>${v}</option>`).join('')}
            </select>
          </div>
          <button class="btn btn-primary" id="f-apply">筛选</button>
        </div>
      </div>
      <section>
        <div id="warning-table"></div>
      </section>
      <section>
        <h3 class="section-title">养殖区溶解氧趋势预览（点击行选择）</h3>
        <div id="trend-preview"><div class="empty-state">选择一条预警记录查看对应养殖区趋势</div></div>
      </section>`;

    const zoneNameOf = (id) => {
      const z = zoneMap[id];
      return z ? z.name : id;
    };
    const tableBox = c.querySelector('#warning-table');
    tableBox.appendChild(WarningTable(list, {
      showActions: true,
      zoneNameOf,
      onChanged: () => warnings.refresh(),
    }));
    tableBox.querySelectorAll('tbody tr').forEach((tr) => {
      tr.style.cursor = 'pointer';
      tr.addEventListener('click', () => {
        const zoneId = tr.dataset.zoneId;
        loadPreview(c, zoneId, zoneMap);
        highlightRow(tr);
      });
    });
  }

  async function loadPreview(c, zoneId, zoneMap) {
    if (!zoneId) return;
    const zone = zoneMap[zoneId];
    try {
      samplesCache = await api(`/api/zones/${encodeURIComponent(zoneId)}/samples?limit=60`);
    } catch (e) {
      samplesCache = [];
    }
    const box = c.querySelector('#trend-preview');
    if (!box) return;
    box.innerHTML = '';
    box.appendChild(WaterTrend(samplesCache, {
      warnThreshold: zone ? zone.do_warn_threshold || 4 : 4,
      dangerThreshold: zone ? zone.do_danger_threshold || 3 : 3,
      title: `${zone ? zone.name : zoneId} 溶解氧趋势`,
    }));
  }

  function highlightRow(tr) {
    const tbody = tr.closest('tbody');
    if (!tbody) return;
    tbody.querySelectorAll('tr').forEach((r) => { r.style.background = ''; });
    tr.style.background = '#e0f2fe';
  }

  function applyBtn() {
    const statusEl = document.getElementById('f-status');
    const typeEl = document.getElementById('f-type');
    if (!statusEl || !typeEl) return;
    filter = { status: statusEl.value, type: typeEl.value, limit: 100 };
    warnings.refresh();
  }

  // Delegated listener for the filter button (re-renders replace the DOM).
  container.addEventListener('click', (e) => {
    if (e.target && e.target.id === 'f-apply') applyBtn();
  });

  return () => {
    unsubWarnings();
    unsubZones();
    warnings.stop();
    zones.stop();
  };
}
