'use strict';

let allModules = [];
let activePath = null;

function injectToast() {
  const toast = document.createElement('div');
  toast.id = 'toast';
  const closeBtn = document.createElement('button');
  closeBtn.id = 'toast-close'; closeBtn.textContent = '×';
  closeBtn.setAttribute('aria-label', 'Close error');
  closeBtn.addEventListener('click', dismissToast);
  const msg = document.createElement('span'); msg.id = 'toast-msg';
  toast.appendChild(closeBtn); toast.appendChild(msg);
  document.body.appendChild(toast);
}

function showToast(message) {
  document.getElementById('toast-msg').textContent = message;
  document.getElementById('toast').classList.add('visible');
}

function dismissToast() {
  const t = document.getElementById('toast');
  if (t) t.classList.remove('visible');
}

// ── Bootstrap ─────────────────────────────────────────────────────────────────

async function init() {
  injectToast();
  const treeEl = document.getElementById('tree');
  const treeSpinner = document.createElement('div');
  treeSpinner.className = 'spinner';
  treeEl.appendChild(treeSpinner);
  try {
    const res = await fetch('/api/modules');
    if (!res.ok) throw new Error('/api/modules returned ' + res.status + ' ' + res.statusText);
    allModules = await res.json();
    treeSpinner.remove();
    renderTree(allModules);
  } catch (err) {
    treeSpinner.remove();
    showToast(err.message);
    return;
  }
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
  const rightEl = document.getElementById('right');
  document.getElementById('detail').style.display = 'none';
  document.getElementById('detail-empty').style.display = 'none';
  const detailSpinner = document.createElement('div');
  detailSpinner.className = 'spinner';
  rightEl.appendChild(detailSpinner);
  try {
    const [specRes, changesRes] = await Promise.all([
      fetch('/api/modules/' + path),
      fetch('/api/modules/' + path + '/changes'),
    ]);
    if (!specRes.ok) throw new Error('/api/modules/' + path + ' returned ' + specRes.status);
    const spec = await specRes.json();
    const changes = changesRes.ok ? await changesRes.json() : [];
    detailSpinner.remove();
    renderDetail(spec, changes);
  } catch (err) {
    detailSpinner.remove();
    showToast(err.message);
    document.getElementById('detail-empty').style.display = '';
  }
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
      if (ch.compiled_commits && ch.compiled_commits.length > 0) {
        const hashes = h('div', 'tl-commits');
        for (const hash of ch.compiled_commits) {
          const tag = h('code', 'tl-commit-hash');
          tag.textContent = hash.slice(0, 7);
          hashes.appendChild(tag);
        }
        body.append(summary, meta, hashes);
      } else {
        body.append(summary, meta);
      }
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

function injectReviewPanel() {
  const panel = document.createElement('div');
  panel.id = 'review-panel';
  panel.style.display = 'none';
  panel.innerHTML =
    '<h2>Pending Review</h2>' +
    '<div id="review-changes"></div>';

  document.body.appendChild(panel);
}

function startReviewPoller() {
  reviewPollerTimer = setInterval(async () => {
    if (!activePath) return;
    try {
      const res = await fetch('/api/modules/' + activePath + '/changes');
      if (!res.ok) {
        hideReviewPanel();
        return;
      }
      const changes = await res.json();
      const draft = changes.filter(c => !c.status || c.status === 'draft');
      if (draft.length === 0) {
        hideReviewPanel();
        return;
      }
      showReviewPanel(draft);
    } catch (_) {
      hideReviewPanel();
    }
  }, 3000);
}

function showReviewPanel(draftChanges) {
  const changesEl = document.getElementById('review-changes');
  changesEl.innerHTML = '';

  for (const change of draftChanges) {
    const card = document.createElement('div');
    card.className = 'review-change-card';

    const heading = document.createElement('div');
    heading.className = 'review-change-heading';
    heading.textContent = (change.id || '') + ' — ' + (change.filename || '') + ' — ' + (change.summary || '');

    if (change.review_comment) {
      const commentNote = document.createElement('div');
      commentNote.className = 'review-comment-note';
      commentNote.textContent = 'Previous feedback: ' + change.review_comment;
      card.appendChild(commentNote);
    }

    const actions = document.createElement('div');
    actions.className = 'review-actions';

    const approveBtn = document.createElement('button');
    approveBtn.className = 'btn-approve';
    approveBtn.textContent = 'Approve';
    approveBtn.addEventListener('click', async () => {
      approveBtn.disabled = true; rejectBtn.disabled = true;
      try {
        const res = await fetch('/api/modules/' + activePath + '/changes/' + change.filename + '/approve', { method: 'POST' });
        if (!res.ok) throw new Error('Approve failed: ' + res.status);
      } catch (err) { showToast(err.message); }
      finally { approveBtn.disabled = false; rejectBtn.disabled = false; }
    });

    const rejectBtn = document.createElement('button');
    rejectBtn.className = 'btn-reject';
    rejectBtn.textContent = 'Reject';
    rejectBtn.addEventListener('click', async () => {
      const comment = prompt('Rejection comment:');
      if (!comment) return;
      approveBtn.disabled = true; rejectBtn.disabled = true;
      try {
        const res = await fetch('/api/modules/' + activePath + '/changes/' + change.filename + '/reject', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ comment }),
        });
        if (!res.ok) throw new Error('Reject failed: ' + res.status);
      } catch (err) { showToast(err.message); }
      finally { approveBtn.disabled = false; rejectBtn.disabled = false; }
    });

    actions.append(approveBtn, rejectBtn);
    card.append(heading, actions);
    changesEl.appendChild(card);
  }

  document.getElementById('review-panel').style.display = 'block';
}

function hideReviewPanel() {
  const panel = document.getElementById('review-panel');
  if (panel) panel.style.display = 'none';
}

// ── Start ─────────────────────────────────────────────────────────────────────

init();
