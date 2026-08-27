// logs.js — 养殖日志 (/logs). Lists daily feeding/death/disease records
// with automatic abnormal-death prompting and a create form. Consumes
// GET /api/logs, GET /api/zones and POST /api/logs.

import { useZones } from '/hooks/use_zones.js';
import { api, post, fmtNum, fmtInt, fmtDate, escapeHtml } from '/api.js';

export function render(container) {
  const zones = useZones(20000);
  zones.start();
  let logs = [];

  async function loadLogs() {
    try {
      logs = await api('/api/logs?limit=100');
    } catch (e) {
      logs = [];
    }
    renderLogs(container, zones.state);
  }
  loadLogs();

  const unsubZones = zones.subscribe((s) => {
    if (s.data) renderLogs(container, s);
  });

  function renderLogs(c, state) {
    const zoneList = (state.data ? state.data.zones || [] : []).map((z) => z.zone).filter(Boolean);
    const zoneMap = {};
    zoneList.forEach((z) => { zoneMap[z.id] = z; });

    c.innerHTML = `
      <div class="page-head">
        <div>
          <h2>养殖日志</h2>
          <p class="muted">每日投喂 / 死亡 / 病害记录（单日死亡超存塘量 1% 自动提示）</p>
        </div>
        <div class="stat-strip">
          <div class="stat"><span class="stat-num">${logs.length}</span><span class="stat-label">记录数</span></div>
          <div class="stat stat-alert"><span class="stat-num">${logs.filter((l) => l.death_abnormal).length}</span><span class="stat-label">异常提示</span></div>
        </div>
      </div>
      <section>
        <h3 class="section-title">录入日志</h3>
        <div class="panel form-card" style="background:var(--panel);border:1px solid var(--line);border-radius:var(--radius);padding:16px;box-shadow:var(--shadow)">
          <div class="form-row">
            <div class="field">
              <label>养殖区</label>
              <select id="log-zone">
                ${zoneList.map((z) => `<option value="${escapeHtml(z.id)}">${escapeHtml(z.name)}</option>`).join('')}
              </select>
            </div>
            <div class="field">
              <label>日期</label>
              <input id="log-date" type="date" value="${today()}" />
            </div>
            <div class="field">
              <label>投喂量 (kg)</label>
              <input id="log-feed" type="number" step="0.1" min="0" value="500" />
            </div>
            <div class="field">
              <label>死亡数</label>
              <input id="log-death" type="number" step="1" min="0" value="0" />
            </div>
            <div class="field">
              <label>病害备注</label>
              <input id="log-note" type="text" placeholder="无" />
            </div>
            <button class="btn btn-primary" id="log-submit">提交</button>
          </div>
        </div>
      </section>
      <section>
        <h3 class="section-title">历史记录</h3>
        <div id="log-table"></div>
      </section>`;

    const tableBox = c.querySelector('#log-table');
    if (!logs.length) {
      tableBox.innerHTML = '<div class="table-wrap"><div class="empty-state">暂无养殖日志</div></div>';
      return;
    }
    const rows = logs.map((l) => {
      const zone = zoneMap[l.zone_id];
      return `
        <tr>
          <td>${fmtDate(l.date)}</td>
          <td>${escapeHtml((zone && zone.name) || l.zone_id)}</td>
          <td class="num">${fmtNum(l.feed_amount, 1)} kg</td>
          <td class="num">${fmtInt(l.death_count)}</td>
          <td>${l.death_abnormal ? '<span class="status-badge status-danger">异常（超存塘量1%）</span>' : '<span class="status-badge status-normal">正常</span>'}</td>
          <td>${escapeHtml(l.disease_note || '-')}</td>
          <td>${escapeHtml(l.operator || '-')}</td>
          <td>${fmtDate(l.created_at)}</td>
        </tr>`;
    }).join('');
    tableBox.innerHTML = `
      <div class="table-wrap">
        <table>
          <thead><tr><th>日期</th><th>养殖区</th><th class="num">投喂量</th><th class="num">死亡数</th><th>异常判定</th><th>病害备注</th><th>操作员</th><th>录入时间</th></tr></thead>
          <tbody>${rows}</tbody>
        </table>
      </div>`;

    c.querySelector('#log-submit').addEventListener('click', async () => {
      const zoneId = c.querySelector('#log-zone').value;
      const date = c.querySelector('#log-date').value;
      const feed = Number(c.querySelector('#log-feed').value || 0);
      const death = Number(c.querySelector('#log-death').value || 0);
      const note = c.querySelector('#log-note').value;
      if (!zoneId || !date) {
        window.alert('请选择养殖区和日期');
        return;
      }
      try {
        const created = await post('/api/logs', {
          zone_id: zoneId, date, feed_amount: feed, death_count: death, disease_note: note,
        });
        window.alert(
          created.death_abnormal
            ? `已记录（⚠ 异常提示：单日死亡 ${created.death_count} 超存塘量 1%）`
            : '已记录 ✓',
        );
        loadLogs();
      } catch (err) {
        window.alert('录入失败：' + err.message);
      }
    });
  }

  return () => {
    unsubZones();
    zones.stop();
  };
}

function today() {
  const d = new Date();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${m}-${day}`;
}
