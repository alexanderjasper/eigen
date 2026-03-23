'use strict';

let allModules = [];
let activePath = null;

// ── Bootstrap ─────────────────────────────────────────────────────────────────

async function init() {
  const res = await fetch('/api/modules');
  allModules = await res.json();
  renderTree(allModules);

  document.getElementById('search').addEventListener('input', e => {
    filterTree(e.target.value.trim().toLowerCase());
  });

  injectReviewPanel();
  startReviewPoller();
}

// ── Tree ──────────────────────────────────────────────────────────────────────

/**
 * Build a nested map from a flat module list.
 * Each node: { name, path, module: ModuleSummary|null, children: Map }
 */
function buildTree(modules) {
  const root = new Map();

  for (const m of modules) {
    const parts = m.path.split('/');
    let cur = root;
    let accumulated = '';
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      accumulated = accumulated ? accumulated + '/' + part : part;
      if (!cur.has(part)) {
        cur.set(part, { name: part, path: accumulated, module: null, children: new Map() });
      }
      if (i === parts.length - 1) {
        cur.get(part).module = m;
      }
      cur = cur.get(part).children;
    }
  }

  return root;
}

function renderTree(modules) {
  const container = document.getElementById('tree');
  container.innerHTML = '';
  const tree = buildTree(modules);
  for (const node of tree.values()) {
    container.appendChild(createNodeEl(node));
  }
}

function createNodeEl(node) {
  const wrapper = document.createElement('div');
  wrapper.className = 'tree-node';
  wrapper.dataset.path = node.path;

  const label = document.createElement('div');
  label.className = 'tree-label';
  if (node.path === activePath) label.classList.add('active');

  const toggle = document.createElement('span');
  toggle.className = 'toggle';

  const name = document.createElement('span');
  name.className = 'tree-name';
  name.textContent = node.name;

  label.appendChild(toggle);
  label.appendChild(name);
  wrapper.appendChild(label);

  const hasChildren = node.children.size > 0;
  let childrenEl = null;

  if (hasChildren) {
    toggle.textContent = '▶';
    toggle.classList.add('open');

    childrenEl = document.createElement('div');
    childrenEl.className = 'tree-children';
    for (const child of node.children.values()) {
      childrenEl.appendChild(createNodeEl(child));
    }
    wrapper.appendChild(childrenEl);

    toggle.addEventListener('click', e => {
      e.stopPropagation();
      const open = toggle.classList.toggle('open');
      childrenEl.classList.toggle('hidden', !open);
    });
  }

  if (node.module) {
    label.addEventListener('click', () => loadDetail(node.path));
  }

  return wrapper;
}

// ── Search ────────────────────────────────────────────────────────────────────

function filterTree(query) {
  if (!query) {
    renderTree(allModules);
    if (activePath) highlightActive(activePath);
    return;
  }

  const filtered = allModules.filter(m =>
    m.path.toLowerCase().includes(query) ||
    (m.title || '').toLowerCase().includes(query)
  );

  // Also include ancestors so the tree structure makes sense.
  const pathSet = new Set(filtered.map(m => m.path));
  for (const m of filtered) {
    const parts = m.path.split('/');
    for (let i = 1; i < parts.length; i++) {
      pathSet.add(parts.slice(0, i).join('/'));
    }
  }

  const visible = allModules.filter(m => pathSet.has(m.path));
  renderTree(visible);
  if (activePath) highlightActive(activePath);
}

function highlightActive(path) {
  const nodes = document.querySelectorAll('.tree-label');
  for (const n of nodes) {
    n.classList.toggle('active', n.closest('.tree-node').dataset.path === path);
  }
}

// ── Detail ────────────────────────────────────────────────────────────────────

async function loadDetail(path) {
  activePath = path;
  highlightActive(path);

  const [specRes, changesRes] = await Promise.all([
    fetch('/api/modules/' + path),
    fetch('/api/modules/' + path + '/changes'),
  ]);

  if (!specRes.ok) return;
  const spec = await specRes.json();
  const changes = changesRes.ok ? await changesRes.json() : [];

  renderDetail(spec, changes);
}

function renderDetail(spec, changes) {
  document.getElementById('detail-empty').style.display = 'none';
  const el = document.getElementById('detail');
  el.style.display = 'block';
  el.innerHTML = '';

  // Title + meta
  const title = h('div', 'detail-title', spec.title || spec.id);
  const metaRow = h('div', 'meta-row');
  metaRow.appendChild(statusBadge(spec.status));
  if (spec.owner) {
    const owner = h('span', 'meta-owner');
    owner.textContent = spec.owner;
    metaRow.appendChild(owner);
  }
  el.appendChild(title);
  el.appendChild(metaRow);

  // Description
  if (spec.description) {
    el.appendChild(section('Description', pre(spec.description)));
  }

  // Behavior
  if (spec.behavior) {
    el.appendChild(section('Behavior', pre(spec.behavior)));
  }

  // Acceptance Criteria
  if (spec.acceptance_criteria && spec.acceptance_criteria.length) {
    const list = h('ul', 'ac-list');
    for (const ac of spec.acceptance_criteria) {
      const item = h('li', 'ac-item');
      const idEl = h('div', 'ac-id'); idEl.textContent = ac.id;
      const desc = h('div', 'ac-desc'); desc.textContent = ac.description;
      const gwt = h('div', 'ac-gwt');
      gwt.innerHTML =
        `<span>Given</span> ${esc(ac.given)}<br>` +
        `<span>When</span> ${esc(ac.when)}<br>` +
        `<span>Then</span> ${esc(ac.then)}`;
      item.append(idEl, desc, gwt);
      list.appendChild(item);
    }
    el.appendChild(section('Acceptance Criteria', list));
  }

  // Dependencies
  if (spec.dependencies && spec.dependencies.length) {
    const list = h('ul', 'dep-list');
    for (const dep of spec.dependencies) {
      const li = document.createElement('li');
      const a = h('a', 'dep-link');
      a.textContent = dep;
      a.href = '#';
      a.addEventListener('click', e => { e.preventDefault(); loadDetail(dep); });
      li.appendChild(a);
      list.appendChild(li);
    }
    el.appendChild(section('Dependencies', list));
  }

  // History
  if (changes && changes.length) {
    const list = h('ul', 'timeline');
    for (const ch of changes) {
      const item = h('li', 'timeline-item');
      const seq = h('span', 'tl-seq'); seq.textContent = String(ch.sequence).padStart(3, '0');
      const body = h('div', 'tl-body');
      const summary = h('div', 'tl-summary'); summary.textContent = ch.summary || ch.type;
      const meta = h('div', 'tl-meta');
      meta.textContent = [ch.timestamp, ch.author].filter(Boolean).join(' · ');
      body.append(summary, meta);
      item.append(seq, body);
      list.appendChild(item);
    }
    el.appendChild(section('History', list));
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function h(tag, className, text) {
  const el = document.createElement(tag);
  if (className) el.className = className;
  if (text) el.textContent = text;
  return el;
}

function pre(text) {
  const el = document.createElement('pre');
  el.className = 'pre-text';
  el.textContent = text;
  return el;
}

function section(title, content) {
  const wrap = h('div', 'section');
  const t = h('div', 'section-title'); t.textContent = title;
  wrap.appendChild(t);
  wrap.appendChild(content);
  return wrap;
}

function statusBadge(status) {
  const badge = h('span', 'badge');
  badge.textContent = status || 'unknown';
  const cls = ['draft', 'stable', 'deprecated'].includes(status)
    ? 'badge-' + status
    : 'badge-unknown';
  badge.classList.add(cls);
  return badge;
}

function esc(str) {
  return (str || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// ── Review Panel ──────────────────────────────────────────────────────────────

let reviewPollerTimer = null;
let activeReviewSessionId = null;

function injectReviewPanel() {
  const panel = document.createElement('div');
  panel.id = 'review-panel';
  panel.style.display = 'none';
  panel.innerHTML =
    '<h2>Spec Review</h2>' +
    '<div id="review-changes"></div>' +
    '<div class="review-actions">' +
      '<button class="btn-approve">Approve</button>' +
      '<button class="btn-reject">Reject</button>' +
    '</div>';

  panel.querySelector('.btn-approve').addEventListener('click', () => {
    submitReview('approved');
  });
  panel.querySelector('.btn-reject').addEventListener('click', () => {
    submitReview('rejected');
  });

  document.body.appendChild(panel);
}

function startReviewPoller() {
  reviewPollerTimer = setInterval(async () => {
    try {
      const res = await fetch('/api/reviews/pending');
      if (res.status === 204 || !res.ok) {
        hideReviewPanel();
        return;
      }
      const data = await res.json();
      showReviewPanel(data.session_id);
    } catch (_) {
      hideReviewPanel();
    }
  }, 3000);
}

async function showReviewPanel(sessionId) {
  try {
    const res = await fetch('/api/reviews/' + sessionId);
    if (!res.ok) {
      hideReviewPanel();
      return;
    }
    const session = await res.json();
    if (session.status === 'submitted') {
      hideReviewPanel();
      return;
    }

    // Already rendering this session — don't rebuild (would reset scroll).
    if (activeReviewSessionId === sessionId &&
        document.getElementById('review-panel').style.display !== 'none') {
      return;
    }

    activeReviewSessionId = sessionId;

    const changesEl = document.getElementById('review-changes');
    changesEl.innerHTML = '';

    for (const change of (session.changes || [])) {
      const card = document.createElement('div');
      card.className = 'review-change-card';

      const heading = document.createElement('div');
      heading.className = 'review-change-heading';
      heading.textContent = change.change_id + ' — ' + change.file_path;

      const yamlPre = document.createElement('pre');
      yamlPre.className = 'review-yaml';
      yamlPre.textContent = change.change_yaml || '';

      const commentLabel = document.createElement('label');
      commentLabel.textContent = 'Comment (optional)';

      const textarea = document.createElement('textarea');
      textarea.className = 'review-comments';
      textarea.dataset.changeId = change.change_id;
      textarea.rows = 3;

      card.append(heading, yamlPre, commentLabel, textarea);
      changesEl.appendChild(card);
    }

    document.getElementById('review-panel').style.display = 'block';
  } catch (_) {
    hideReviewPanel();
  }
}

function hideReviewPanel() {
  const panel = document.getElementById('review-panel');
  if (panel) panel.style.display = 'none';
  activeReviewSessionId = null;
}

async function submitReview(decision) {
  if (!activeReviewSessionId) return;

  const changeComments = {};
  const textareas = document.querySelectorAll('.review-comments');
  for (const ta of textareas) {
    if (ta.value.trim()) {
      changeComments[ta.dataset.changeId] = ta.value.trim();
    }
  }

  try {
    const res = await fetch('/api/reviews/' + activeReviewSessionId + '/submit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ decision, change_comments: changeComments }),
    });
    if (res.ok) {
      hideReviewPanel();
    }
  } catch (_) {
    // ignore, poller will retry
  }
}

// ── Start ─────────────────────────────────────────────────────────────────────

init().catch(err => {
  document.getElementById('detail-empty').textContent = 'Failed to load: ' + err.message;
});
