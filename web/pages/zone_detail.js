// zone_detail.js — 养殖区详情 (/zones/{id}). Uses ZoneCard + WaterTrend +
// WarningTable + useZones + useWarnings. Consumes
// GET /api/overview, /api/zones/{id}/samples, /api/warnings, /api/aeration
// and the restore/aerate actions.

import { useZones } from '/hooks/use_zones.js';
import { useWarnings } from '/hooks/use_warnings.js';
import { ZoneCard } from '/components/zone_card.js';
import { WaterTrend } from '/components/water_trend.js';
import { WarningTable } from '/components/warning_table.js';
import { api, post, fmtTime, fmtNum, escapeHtml } from '/api.js';
import { AERATOR_STATUS_LABEL, AERATOR_STATUS_CLASS, AERATION_ACTION_LABEL, FEEDBACK_LABEL } from '/enums.js';

export function render(container, params) {
  const zoneId = params.id;
  let samples = [];
  let aeration = [];
  let currentZo = null;

  const zones = useZones(6000);
  const warnings = useWarnings({ zone_id: zoneId, limit: 50 }, 8000);
  zones.start();
  warnings.start();

  async function loadTrend() {
    try {
      samples = await api(`/api/zones/${encodeURIComponent(zoneId)}/samples?limit=120`);
      aeration = await api('/api/aeration?limit=50');
      renderDetail();
    } catch (e) {
      // keep last view; next poll retries
      console.error('zone_detail: trend load failed', e);
    }
  }
  loadTrend();

  const unsubZones = zones.subscribe((s) => {
    if (s.data) {
      currentZo = (s.data.zones || []).find((z) => z.zone && z.zone.id === zoneId) || null;
      renderDetail();
    }
  });
  const unsubWarnings = warnings.subscribe(() => renderDetail());

  function renderDetail() {
    if (!currentZo) {
      container.innerHTML = '<div class="page-loading">加载养殖区详情…</div>';
      return;
    }
    const zo = currentZo;
    const zone = zo.zone;
    const aer = aeration.filter((l) => l.zone_id === zoneId).slice(0, 8);
    const canRestore = zone.status === 'aerating' && zone.restore_eligible;

    container.innerHTML = `
      <div class="page-head">
        <div>
          <h2>养殖区详情</h2>
          <p class="muted">${escapeHtml(zone.id)} · ${fmtTime(zone.status_since)} 进入当前状态</p>
        </div>
        <div>
          <button class="btn btn-primary" id="btn-aerate" ${zone.status === 'aerating' ? 'disabled' : ''}>启动增氧</button>
          <button class="btn btn-danger" id="btn-restore" ${canRestore ? '' : 'disabled'}>确认恢复</button>
        </div>
      </div>
      <div class="detail-grid">
        <div id="zone-card"></div>
        <div class="split">
          <div id="trend"></div>
          <div id="aeration-list"></div>
        </div>
      </div>
      <section>
        <h3 class="section-title">预警记录</h3>
        <div id="warnings"></div>
      </section>`;

    container.querySelector('#zone-card').appendChild(ZoneCard(zo, { compact: true }));

    if (canRestore) {
      const notice = document.createElement('div');
      notice.className = 'notice notice-ok';
      notice.innerHTML = '✓ 溶解氧已恢复至 5 mg/L 以上并持续满足恢复条件，可确认恢复。';
      container.querySelector('#zone-card').appendChild(notice);
    }

    // WaterTrend (shared component) fed with the zone's samples.
    const trendBox = container.querySelector('#trend');
    const trendSamples = (samples || []).slice().sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
    trendBox.appendChild(WaterTrend(trendSamples, {
      warnThreshold: zone.do_warn_threshold || 4,
      dangerThreshold: zone.do_danger_threshold || 3,
      title: '溶解氧趋势 (mg/L)',
    }));

    // Aeration timeline.
    const aerBox = container.querySelector('#aeration-list');
    if (!aer.length) {
      aerBox.innerHTML = '<div class="trend-panel"><div class="empty-state">暂无增氧联动记录</div></div>';
    } else {
      const rows = aer.map((l) => `
        <tr>
          <td>${AERATION_ACTION_LABEL[l.action] || l.action}</td>
          <td><span class="status-badge ${AERATOR_STATUS_CLASS[l.status] || ''}">${AERATOR_STATUS_LABEL[l.status] || l.status}</span></td>
          <td>${FEEDBACK_LABEL[l.feedback] || l.feedback}</td>
          <td>${fmtTime(l.command_time)}</td>
          <td class="log-line" title="${escapeHtml(l.note || '')}">${escapeHtml((l.note || '').slice(0, 18))}</td>
        </tr>`).join('');
      aerBox.innerHTML = `
        <div class="trend-panel">
          <strong>增氧联动</strong>
          <div class="table-wrap" style="margin-top:8px">
            <table>
              <thead><tr><th>动作</th><th>状态</th><th>反馈</th><th>下发时间</th><th>说明</th></tr></thead>
              <tbody>${rows}</tbody>
            </table>
          </div>
        </div>`;
    }

    const warnBox = container.querySelector('#warnings');
    const warnList = (warnings.state.data || []).slice();
    warnBox.appendChild(WarningTable(warnList, { zoneNameOf: () => zone.name, onChanged: () => warnings.refresh() }));

    container.querySelector('#btn-aerate').addEventListener('click', async (e) => {
      const btn = e.currentTarget;
      btn.disabled = true;
      try {
        await post(`/api/zones/${encodeURIComponent(zoneId)}/aerate`);
        zones.refresh();
        loadTrend();
      } catch (err) {
        window.alert('启动增氧失败：' + err.message);
        btn.disabled = false;
      }
    });
    container.querySelector('#btn-restore').addEventListener('click', async (e) => {
      const btn = e.currentTarget;
      btn.disabled = true;
      try {
        await post(`/api/zones/${encodeURIComponent(zoneId)}/restore`);
        zones.refresh();
        loadTrend();
      } catch (err) {
        window.alert('恢复确认失败：' + err.message);
        btn.disabled = false;
      }
    });
  }

  return () => {
    unsubZones();
    unsubWarnings();
    zones.stop();
    warnings.stop();
  };
}
