// aeration.js — 增氧控制 (/aeration). Lists every zone's aerator status,
// allows manual start and device feedback, and shows the recent aeration
// command log. Consumes GET /api/overview, GET /api/aeration,
// POST /api/zones/{id}/aerate, POST /api/aeration/{id}/feedback.

import { useZones } from '/hooks/use_zones.js';
import { api, post, fmtTime, escapeHtml } from '/api.js';
import {
  AERATOR_STATUS_LABEL, AERATOR_STATUS_CLASS, AERATION_ACTION_LABEL,
  FEEDBACK_LABEL, ZONE_STATUS_LABEL, ZONE_STATUS_CLASS,
} from '/enums.js';

export function render(container) {
  const zones = useZones(5000);
  zones.start();
  let aeration = [];

  async function loadLogs() {
    try {
      aeration = await api('/api/aeration?limit=100');
    } catch (e) {
      aeration = [];
    }
    renderAeration(container, zones.state);
  }
  loadLogs();

  const unsubZones = zones.subscribe((s) => renderAeration(container, s));

  function renderAeration(c, state) {
    if (state.loading && !state.data) {
      c.innerHTML = '<div class="page-loading">加载增氧控制…</div>';
      return;
    }
    const data = state.data || { zones: [] };
    const zoneList = data.zones || [];

    c.innerHTML = `
      <div class="page-head">
        <div>
          <h2>增氧控制</h2>
          <p class="muted">增氧机状态 · 手动启停 · 设备反馈（超时未反馈按故障处理）</p>
        </div>
        <div class="stat-strip">
          <div class="stat stat-aerate"><span class="stat-num">${data.totals ? data.totals.active_aerators ?? 0 : 0}</span><span class="stat-label">运行增氧机</span></div>
        </div>
      </div>
      <section>
        <h3 class="section-title">增氧机状态</h3>
        <div class="zone-grid" id="aer-grid"></div>
      </section>
      <section>
        <h3 class="section-title">联动记录</h3>
        <div id="aer-log"></div>
      </section>`;

    const grid = c.querySelector('#aer-grid');
    for (const zo of zoneList) {
      grid.appendChild(aeratorCard(zo, aeration, () => {
        zones.refresh();
        loadLogs();
      }));
    }
    if (!zoneList.length) {
      grid.innerHTML = '<div class="empty-state">暂无养殖区</div>';
    }

    const logBox = c.querySelector('#aer-log');
    if (!aeration.length) {
      logBox.innerHTML = '<div class="table-wrap"><div class="empty-state">暂无增氧联动记录</div></div>';
      return;
    }
    const rows = aeration.map((l) => `
      <tr>
        <td>${escapeHtml(l.zone_id)}</td>
        <td>${escapeHtml(l.aerator_id)}</td>
        <td>${AERATION_ACTION_LABEL[l.action] || l.action}</td>
        <td><span class="status-badge ${AERATOR_STATUS_CLASS[l.status] || ''}">${AERATOR_STATUS_LABEL[l.status] || l.status}</span></td>
        <td>${FEEDBACK_LABEL[l.feedback] || l.feedback}</td>
        <td>${fmtTime(l.command_time)}</td>
        <td>${escapeHtml(l.trigger || '-')}</td>
        <td class="log-line" title="${escapeHtml(l.note || '')}">${escapeHtml((l.note || '').slice(0, 20))}</td>
      </tr>`).join('');
    logBox.innerHTML = `
      <div class="table-wrap">
        <table>
          <thead><tr><th>养殖区</th><th>增氧机</th><th>动作</th><th>状态</th><th>反馈</th><th>下发时间</th><th>触发</th><th>说明</th></tr></thead>
          <tbody>${rows}</tbody>
        </table>
      </div>`;
  }

  function aeratorCard(zo, logs, onChanged) {
    const zone = zo.zone || {};
    const status = zo.aerator_status || 'stopped';
    const action = zo.aerator_action || '';
    const lastLog = logs.filter((l) => l.zone_id === zone.id)[0] || null;
    const card = document.createElement('article');
    card.className = 'zone-card';
    card.innerHTML = `
      <div class="zone-card-head">
        <div>
          <h3 class="zone-name">${escapeHtml(zone.name || zone.id)}</h3>
          <div class="zone-sub">${escapeHtml(zone.id)} · 状态 ${ZONE_STATUS_LABEL[zone.status] || zone.status}</div>
        </div>
        <span class="status-badge ${AERATOR_STATUS_CLASS[status] || 'status-stopped'}">${AERATOR_STATUS_LABEL[status] || status}</span>
      </div>
      <div class="zone-card-body">
        <div class="metrics">
          <div class="metric"><span class="metric-label">溶解氧</span><span class="metric-value">${zo.latest_do != null && zo.latest_do >= 0 ? zo.latest_do.toFixed(2) : '-'}</span></div>
          <div class="metric"><span class="metric-label">动作</span><span class="metric-value" style="font-size:14px">${action ? AERATION_ACTION_LABEL[action] : '-'}</span></div>
          <div class="metric"><span class="metric-label">反馈</span><span class="metric-value" style="font-size:14px">${lastLog ? FEEDBACK_LABEL[lastLog.feedback] : '-'}</span></div>
        </div>
        <div class="zone-card-foot">
          <button class="btn btn-sm btn-primary act-start" ${status === 'starting' || status === 'running' ? 'disabled' : ''}>启动</button>
          <button class="btn btn-sm act-stop" ${(status === 'stopped' || status === 'stopping') ? 'disabled' : ''}>停止</button>
          <button class="btn btn-sm act-ack" data-fb="acknowledged">应答</button>
          <button class="btn btn-sm act-fb" data-fb="started" ${status !== 'starting' ? 'disabled' : ''}>反馈已启动</button>
          <button class="btn btn-sm act-fb" data-fb="stopped" ${status !== 'stopping' ? 'disabled' : ''}>反馈已停止</button>
          <button class="btn btn-sm act-fb" data-fb="fault">反馈故障</button>
        </div>
      </div>`;

    card.querySelector('.act-start').addEventListener('click', async (e) => {
      const btn = e.currentTarget;
      btn.disabled = true;
      try {
        const log = await post(`/api/zones/${encodeURIComponent(zone.id)}/aerate`);
        window.alert('已下发启动指令：' + log.id);
        onChanged();
      } catch (err) {
        window.alert('启动失败：' + err.message);
        btn.disabled = false;
      }
    });
    card.querySelector('.act-stop').addEventListener('click', async () => {
      try {
        const log = await post(`/api/zones/${encodeURIComponent(zone.id)}/stop-aeration`);
        window.alert('已下发停止指令：' + log.id);
        onChanged();
      } catch (err) {
        window.alert('停止失败：' + err.message);
      }
    });
    // Feedback buttons need the latest command id of the zone.
    const latest = logs.find((l) => l.zone_id === zone.id);
    card.querySelectorAll('.act-fb, .act-ack').forEach((btn) => {
      btn.addEventListener('click', async (e) => {
        const fb = e.currentTarget.dataset.fb;
        if (!latest) {
          window.alert('该养殖区暂无增氧指令');
          return;
        }
        try {
          const log = await post(`/api/aeration/${encodeURIComponent(latest.id)}/feedback`, { feedback: fb });
          window.alert(`反馈 ${fb} 已记录：${log.status}`);
          onChanged();
        } catch (err) {
          window.alert('反馈失败：' + err.message);
        }
      });
    });
    return card;
  }

  return () => {
    unsubZones();
    zones.stop();
  };
}
