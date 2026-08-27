// app.js — SPA router. Maps pathnames to pages and wires navigation.

import * as overview from '/pages/overview.js';
import * as zoneDetail from '/pages/zone_detail.js';
import * as warnings from '/pages/warnings.js';
import * as aeration from '/pages/aeration.js';
import * as logs from '/pages/logs.js';

const routes = [
  { pattern: /^\/(?:index\.html)?$/, page: overview, title: '牧场总览' },
  { pattern: /^\/zones\/([^/]+)$/, page: zoneDetail, title: '养殖区详情' },
  { pattern: /^\/warnings$/, page: warnings, title: '预警台' },
  { pattern: /^\/aeration$/, page: aeration, title: '增氧控制' },
  { pattern: /^\/logs$/, page: logs, title: '养殖日志' },
];

const app = document.getElementById('app');
let currentCleanup = null;
let navLinks = [];

function route() {
  const path = window.location.pathname;
  for (const r of routes) {
    const m = path.match(r.pattern);
    if (m) {
      if (currentCleanup) {
        try { currentCleanup(); } catch (e) { console.error('cleanup error', e); }
        currentCleanup = null;
      }
      document.title = r.title + ' · 海洋牧场养殖环境监测';
      const params = m.length > 1 ? { id: decodeURIComponent(m[1]) } : {};
      currentCleanup = r.page.render(app, params);
      highlightNav(path);
      return;
    }
  }
  app.innerHTML = '<div class="error-state">404 · 页面不存在</div>';
}

function highlightNav(path) {
  for (const a of navLinks) {
    const href = a.getAttribute('href');
    a.classList.toggle('active', path === href || (href !== '/' && path.startsWith(href)));
  }
}

document.addEventListener('click', (e) => {
  const a = e.target.closest('a[data-link]');
  if (!a) return;
  const href = a.getAttribute('href');
  if (!href || href.startsWith('http') || href.startsWith('//')) return;
  e.preventDefault();
  if (window.location.pathname !== href) {
    window.history.pushState({}, '', href);
    route();
  } else {
    route();
  }
});

window.addEventListener('popstate', route);

function init() {
  navLinks = Array.from(document.querySelectorAll('#main-nav .nav-link'));
  setInterval(() => {
    const clock = document.getElementById('clock');
    if (clock) clock.textContent = new Date().toLocaleTimeString('zh-CN', { hour12: false });
  }, 1000);
  pollHealth();
  route();
}

async function pollHealth() {
  const pill = document.getElementById('health-pill');
  try {
    const res = await fetch('/api/healthz');
    const ok = res.ok;
    pill.textContent = ok ? '● 服务正常' : '● 异常';
    pill.classList.toggle('health-ok', ok);
    pill.classList.toggle('health-bad', !ok);
  } catch (e) {
    pill.textContent = '● 离线';
    pill.classList.remove('health-ok');
    pill.classList.add('health-bad');
  }
  setTimeout(pollHealth, 15000);
}

init();
