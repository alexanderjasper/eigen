'use strict';

let allModules = [];
let allWorktrees = [];
let activePath = null;
let activeWorktreeName = null;

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
    const [worktreesRes, modulesRes] = await Promise.all([
      fetch('/api/worktrees'),
      fetch('/api/modules'),
    ]);
    if (!modulesRes.ok) throw new Error('/api/modules returned ' + modulesRes.status + ' ' + modulesRes.statusText);
    allModules = await modulesRes.json();
    allWorktrees = worktreesRes.ok ? await worktreesRes.json() : [];
    treeSpinner.remove();
    renderTreeView();
    updatePageTitle();
  } catch (err) {
    treeSpinner.remove();
    showToast(err.message);
    return;
  }
  document.getElementById('search').addEventListener('input', e => {
    filterTree(e.target.value.trim().toLowerCase());
  });
  injectReviewPanel();
  initResizeHandles();
  startReviewPoller();
  startWorktreePoller();
}

function updatePageTitle() {
  document.title = allWorktrees.length > 1
    ? 'eigen (' + allWorktrees.length + ' branches)'
    : 'eigen';
}

function renderTreeView() {
  if (allWorktrees.length > 1) {
    renderGroupedTree();
  } else {
    renderTree(allModules);
  }
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

function getVisibleLabels() {
  return Array.from(document.querySelectorAll('.tree-label')).filter(el =>
    !el.closest('.tree-children.hidden')
  );
}

function getParentLabel(label) {
  const parentChildren = label.closest('.tree-node').parentElement;
  if (!parentChildren || !parentChildren.classList.contains('tree-children')) return null;
  return parentChildren.closest('.tree-node').querySelector(':scope > .tree-label');
}

function moveFocus(label) {
  if (!label) return;
  for (const l of document.querySelectorAll('.tree-label')) l.setAttribute('tabindex', '-1');
  label.setAttribute('tabindex', '0');
  label.focus();
}

function initRovingTabindex() {
  const labels = getVisibleLabels();
  labels.forEach((l, i) => l.setAttribute('tabindex', i === 0 ? '0' : '-1'));
}

function renderTree(modules) {
  const container = document.getElementById('tree');
  container.innerHTML = '';
  container.setAttribute('role', 'tree');
  const tree = buildTree(modules);
  for (const node of tree.values()) {
    container.appendChild(createNodeEl(node, null));
  }
  initRovingTabindex();
  container.addEventListener('keydown', e => {
    if (!e.target.classList.contains('tree-label')) return;
    const label = e.target;
    const visible = getVisibleLabels();
    const idx = visible.indexOf(label);

    switch (e.key) {
      case 'ArrowDown': {
        e.preventDefault();
        moveFocus(visible[idx + 1] ?? null);
        break;
      }
      case 'ArrowUp': {
        e.preventDefault();
        moveFocus(visible[idx - 1] ?? null);
        break;
      }
      case 'ArrowRight': {
        e.preventDefault();
        const expanded = label.getAttribute('aria-expanded');
        if (expanded === 'false') {
          // expand
          label.closest('.tree-node').querySelector(':scope > .tree-children')?.classList.remove('hidden');
          label.closest('.tree-node').querySelector(':scope > .toggle')?.classList.add('open');
          label.setAttribute('aria-expanded', 'true');
        } else if (expanded === 'true') {
          // move to first child
          const newVisible = getVisibleLabels();
          moveFocus(newVisible[newVisible.indexOf(label) + 1] ?? null);
        }
        // null (leaf) — no-op
        break;
      }
      case 'ArrowLeft': {
        e.preventDefault();
        const expanded = label.getAttribute('aria-expanded');
        if (expanded === 'true') {
          // collapse
          label.closest('.tree-node').querySelector(':scope > .tree-children')?.classList.add('hidden');
          label.closest('.tree-node').querySelector(':scope > .toggle')?.classList.remove('open');
          label.setAttribute('aria-expanded', 'false');
        } else {
          // move to parent
          moveFocus(getParentLabel(label));
        }
        break;
      }
      case 'Enter': {
        e.preventDefault();
        label.click();
        break;
      }
    }
  });
}

function createNodeEl(node, worktreeName) {
  const wrapper = document.createElement('div');
  wrapper.className = 'tree-node';
  wrapper.dataset.path = node.path;
  if (worktreeName) wrapper.dataset.worktree = worktreeName;

  const label = document.createElement('div');
  label.className = 'tree-label';
  label.setAttribute('role', 'treeitem');
  label.setAttribute('tabindex', '-1');
  if (node.path === activePath && (worktreeName || null) === activeWorktreeName) label.classList.add('active');

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
    label.setAttribute('aria-expanded', 'true');

    childrenEl = document.createElement('div');
    childrenEl.className = 'tree-children';
    for (const child of node.children.values()) {
      childrenEl.appendChild(createNodeEl(child, worktreeName));
    }
    wrapper.appendChild(childrenEl);

    toggle.addEventListener('click', e => {
      e.stopPropagation();
      const open = toggle.classList.toggle('open');
      childrenEl.classList.toggle('hidden', !open);
      label.setAttribute('aria-expanded', String(open));
    });
  }

  if (node.module) {
    label.addEventListener('click', () => loadDetail(node.path, node.module.worktree));
  }

  return wrapper;
}

// ── Grouped tree (multi-worktree) ─────────────────────────────────────────────

function renderGroupedTree() {
  const container = document.getElementById('tree');
  container.innerHTML = '';
  container.setAttribute('role', 'tree');

  for (const wt of allWorktrees) {
    const wtModules = allModules.filter(m => m.worktree === wt.name);
    const groupSection = document.createElement('div');
    groupSection.className = 'tree-group-section';
    groupSection.dataset.worktree = wt.name;

    // Group header.
    const header = document.createElement('div');
    header.className = 'tree-group-header tree-label';
    header.setAttribute('role', 'treeitem');
    header.setAttribute('aria-expanded', 'true');
    header.setAttribute('tabindex', '-1');

    const toggle = document.createElement('span');
    toggle.className = 'toggle open';
    toggle.textContent = '▶';

    const nameEl = document.createElement('span');
    nameEl.className = 'tree-name';
    nameEl.textContent = wt.name + ' (' + wt.branch + ')';

    header.appendChild(toggle);
    header.appendChild(nameEl);
    groupSection.appendChild(header);

    // Children container.
    const childrenEl = document.createElement('div');
    childrenEl.className = 'tree-group-children tree-children';
    const tree = buildTree(wtModules);
    for (const node of tree.values()) {
      childrenEl.appendChild(createNodeEl(node, wt.name));
    }
    groupSection.appendChild(childrenEl);

    // Toggle on click.
    toggle.addEventListener('click', e => {
      e.stopPropagation();
      const open = toggle.classList.toggle('open');
      childrenEl.classList.toggle('hidden', !open);
      header.setAttribute('aria-expanded', String(open));
    });
    header.addEventListener('click', e => {
      if (e.target === toggle) return;
      const open = toggle.classList.toggle('open');
      childrenEl.classList.toggle('hidden', !open);
      header.setAttribute('aria-expanded', String(open));
    });

    container.appendChild(groupSection);
  }

  initRovingTabindex();
}

// ── Search ────────────────────────────────────────────────────────────────────

function filterTree(query) {
  if (!query) {
    renderTreeView();
    if (activePath) highlightActive(activePath, activeWorktreeName);
    return;
  }

  if (allWorktrees.length > 1) {
    filterGroupedTree(query);
    if (activePath) highlightActive(activePath, activeWorktreeName);
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
  if (activePath) highlightActive(activePath, activeWorktreeName);
}

function filterGroupedTree(query) {
  renderGroupedTree();
  // Hide group sections where no child labels match.
  const sections = document.querySelectorAll('.tree-group-section');
  for (const section of sections) {
    const childLabels = Array.from(section.querySelectorAll('.tree-node .tree-label'));
    const anyMatch = childLabels.some(lbl => {
      const path = lbl.closest('.tree-node').dataset.path || '';
      const name = lbl.querySelector('.tree-name')?.textContent || '';
      return path.toLowerCase().includes(query) || name.toLowerCase().includes(query);
    });
    if (!anyMatch) {
      section.style.display = 'none';
    } else {
      section.style.display = '';
      // Hide non-matching child nodes.
      for (const node of section.querySelectorAll('.tree-node')) {
        const path = node.dataset.path || '';
        const name = node.querySelector(':scope > .tree-label .tree-name')?.textContent || '';
        if (!path.toLowerCase().includes(query) && !name.toLowerCase().includes(query)) {
          node.style.display = 'none';
        } else {
          node.style.display = '';
        }
      }
    }
  }
}

function highlightActive(path, worktreeName) {
  const nodes = document.querySelectorAll('.tree-label');
  for (const n of nodes) {
    const treeNode = n.closest('.tree-node');
    if (!treeNode) continue; // group headers have no .tree-node ancestor
    const pathMatch = treeNode.dataset.path === path;
    const wtMatch = worktreeName == null
      ? !treeNode.dataset.worktree  // flat mode: no worktree set
      : treeNode.dataset.worktree === worktreeName;
    n.classList.toggle('active', pathMatch && wtMatch);
  }
}

// ── Detail ────────────────────────────────────────────────────────────────────

async function loadDetail(path, worktreeName) {
  activePath = path;
  activeWorktreeName = worktreeName || null;
  highlightActive(path, activeWorktreeName);
  const rightEl = document.getElementById('right');
  document.getElementById('detail').style.display = 'none';
  document.getElementById('detail-empty').style.display = 'none';
  const detailSpinner = document.createElement('div');
  detailSpinner.className = 'spinner';
  rightEl.appendChild(detailSpinner);
  const wtParam = worktreeName ? '?worktree=' + encodeURIComponent(worktreeName) : '';
  try {
    const [specRes, changesRes] = await Promise.all([
      fetch('/api/modules/' + path + wtParam),
      fetch('/api/modules/' + path + '/changes' + wtParam),
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
  if (spec.deprecation_reason) {
    const depReason = h('div', 'deprecation-reason');
    depReason.textContent = 'Deprecated: ' + spec.deprecation_reason;
    metaRow.appendChild(depReason);
  }
  el.appendChild(title);

  // Worktree/branch secondary label (AC-042).
  if (spec.worktree && spec.branch) {
    const wtLabel = h('div', 'detail-worktree-label');
    wtLabel.textContent = spec.worktree + ' (' + spec.branch + ')';
    el.appendChild(wtLabel);
  }

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
  const cls = ['draft', 'stable', 'approved', 'compiled', 'deprecated', 'removed'].includes(status)
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
let reviewDraftChanges = [];
let reviewDetailIndex = null;

function injectReviewPanel() {
  const panel = document.createElement('div');
  panel.id = 'review-panel';
  panel.style.display = 'none';
  panel.innerHTML =
    '<h2>Pending Review</h2>' +
    '<div id="review-changes"></div>';
  document.body.appendChild(panel);
}

function stopReviewPoller() {
  if (reviewPollerTimer !== null) {
    clearTimeout(reviewPollerTimer);
    reviewPollerTimer = null;
  }
}

async function pollOnce() {
  try {
    const res = await fetch('/api/changes?status=draft');
    if (!res.ok) { hideReviewPanel(); schedulePoll(); return; }
    const allDrafts = await res.json();
    // Same change file appears once per worktree — deduplicate by module_path+filename,
    // preferring the main/primary worktree so approve/reject targets the canonical copy.
    const seen = new Map();
    for (const d of allDrafts) {
      const key = (d.module_path || '') + '/' + (d.filename || '');
      if (!seen.has(key) || !d.worktree || d.worktree === 'main') seen.set(key, d);
    }
    const drafts = Array.from(seen.values());
    if (drafts.length === 0) { hideReviewPanel(); schedulePoll(); return; }
    // Don't interrupt if the panel is already open.
    const panel = document.getElementById('review-panel');
    if (panel && panel.style.display !== 'none') { schedulePoll(); return; }
    showReviewPanel(drafts);
  } catch (_) {
    hideReviewPanel();
    schedulePoll();
  }
}

function schedulePoll() {
  reviewPollerTimer = setTimeout(pollOnce, 3000);
}

function startReviewPoller() {
  stopReviewPoller();
  schedulePoll();
}

function showReviewPanel(draftChanges) {
  stopReviewPoller();
  reviewDraftChanges = draftChanges;
  reviewDetailIndex = null;

  const panel = document.getElementById('review-panel');
  const savedWidth = localStorage.getItem('eigen-review-width');
  if (savedWidth) panel.style.width = savedWidth + 'px';
  panel.style.display = 'block';
  renderReviewList();
}

function renderReviewList() {
  reviewDetailIndex = null;
  const changesEl = document.getElementById('review-changes');
  changesEl.innerHTML = '';

  const header = document.createElement('div');
  header.className = 'review-list-header';

  const count = document.createElement('div');
  count.className = 'review-list-count';
  const n = reviewDraftChanges.length;
  count.textContent = n + ' change' + (n !== 1 ? 's' : '') + ' pending';

  const approveAllBtn = document.createElement('button');
  approveAllBtn.className = 'btn-approve-all';
  approveAllBtn.textContent = 'Approve All';
  approveAllBtn.addEventListener('click', () => approveAll(approveAllBtn, count));

  header.appendChild(count);
  header.appendChild(approveAllBtn);
  changesEl.appendChild(header);

  reviewDraftChanges.forEach((change, idx) => {
    const row = document.createElement('div');
    row.className = 'review-list-row';
    row.setAttribute('role', 'button');
    row.setAttribute('tabindex', '0');

    const modEl = document.createElement('span');
    modEl.className = 'review-list-module';
    modEl.textContent = change.module_path || '';

    const summaryEl = document.createElement('span');
    summaryEl.className = 'review-list-summary';
    summaryEl.textContent = change.summary || change.filename || '';

    row.append(modEl, summaryEl, statusBadge(change.status || 'draft'));
    row.addEventListener('click', () => renderReviewDetail(idx));
    row.addEventListener('keydown', e => {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); renderReviewDetail(idx); }
    });
    changesEl.appendChild(row);
  });
}

async function approveAll(btn, countEl) {
  btn.disabled = true;
  const total = reviewDraftChanges.length;
  let approved = 0;
  const snapshot = reviewDraftChanges.slice();

  for (const change of snapshot) {
    btn.textContent = 'Approving ' + (approved + 1) + '/' + total + '...';
    const wtParam = change.worktree ? '?worktree=' + encodeURIComponent(change.worktree) : '';
    try {
      const res = await fetch(
        '/api/modules/' + (change.module_path||'') + '/changes/' + (change.filename||'') + '/approve' + wtParam,
        { method: 'POST' }
      );
      if (!res.ok) throw new Error('Approve failed: ' + res.status);
      const liveIdx = reviewDraftChanges.indexOf(change);
      if (liveIdx !== -1) reviewDraftChanges.splice(liveIdx, 1);
      approved++;
    } catch (err) {
      showToast(err.message);
      break;
    }
  }

  if (reviewDraftChanges.length === 0) { hideReviewPanel(); return; }
  btn.disabled = false;
  btn.textContent = 'Approve All';
  renderReviewList();
}

function renderReviewDetail(index) {
  reviewDetailIndex = index;
  const change = reviewDraftChanges[index];
  const changesEl = document.getElementById('review-changes');
  changesEl.innerHTML = '';

  // Navigation bar
  const nav = document.createElement('div');
  nav.className = 'review-nav';

  const backBtn = document.createElement('button');
  backBtn.className = 'review-back-link';
  backBtn.textContent = '← Back to list';
  backBtn.addEventListener('click', renderReviewList);

  const spacer = document.createElement('span');
  spacer.className = 'review-nav-spacer';

  const prevBtn = document.createElement('button');
  prevBtn.className = 'review-nav-btn';
  prevBtn.textContent = '← Prev';
  prevBtn.disabled = index === 0;
  prevBtn.addEventListener('click', () => renderReviewDetail(index - 1));

  const nextBtn = document.createElement('button');
  nextBtn.className = 'review-nav-btn';
  nextBtn.textContent = 'Next →';
  nextBtn.disabled = index === reviewDraftChanges.length - 1;
  nextBtn.addEventListener('click', () => renderReviewDetail(index + 1));

  nav.append(backBtn, spacer, prevBtn, nextBtn);
  changesEl.appendChild(nav);

  // Card
  const card = document.createElement('div');
  card.className = 'review-change-card';

  const heading = document.createElement('div');
  heading.className = 'review-change-heading';
  heading.textContent = (change.module_path || '') + ' / ' + (change.filename || '') + ' — ' + (change.summary || '');
  card.appendChild(heading);

  // Rich change fields (no raw YAML)
  if (change.changes && typeof change.changes === 'object') {
    const fieldsEl = document.createElement('div');
    fieldsEl.className = 'review-fields';
    renderChangeFields(fieldsEl, change.changes);
    card.appendChild(fieldsEl);
  }

  // Comment label
  const commentLabelEl = document.createElement('div');
  commentLabelEl.style.cssText = 'font-size:12px;color:var(--text-muted);margin-top:14px;margin-bottom:4px;';
  commentLabelEl.textContent = 'Comment';
  card.appendChild(commentLabelEl);

  // Always-visible comment textarea
  const textarea = document.createElement('textarea');
  textarea.className = 'review-comments';
  textarea.rows = 3;
  textarea.placeholder = 'Add a comment (optional for approve, required for reject)...';
  if (change.review_comment) textarea.value = change.review_comment;
  card.appendChild(textarea);

  const validation = document.createElement('div');
  validation.className = 'review-validation';
  validation.textContent = 'A comment is required to reject.';
  card.appendChild(validation);

  // Actions
  const wtParam = change.worktree
    ? '?worktree=' + encodeURIComponent(change.worktree)
    : '';
  const modPath = change.module_path || '';
  const filename = change.filename || '';

  const actions = document.createElement('div');
  actions.className = 'review-actions';

  const approveBtn = document.createElement('button');
  approveBtn.className = 'btn-approve';
  approveBtn.textContent = 'Approve';
  approveBtn.addEventListener('click', async () => {
    approveBtn.disabled = true; rejectBtn.disabled = true;
    const comment = textarea.value.trim();
    const body = comment ? JSON.stringify({ comment }) : undefined;
    try {
      const res = await fetch('/api/modules/' + modPath + '/changes/' + filename + '/approve' + wtParam, {
        method: 'POST',
        headers: body ? { 'Content-Type': 'application/json' } : {},
        body,
      });
      if (!res.ok) throw new Error('Approve failed: ' + res.status);
      reviewDraftChanges.splice(index, 1);
      if (reviewDraftChanges.length === 0) { hideReviewPanel(); return; }
      renderReviewList();
    } catch (err) {
      showToast(err.message);
      approveBtn.disabled = false; rejectBtn.disabled = false;
    }
  });

  const rejectBtn = document.createElement('button');
  rejectBtn.className = 'btn-reject';
  rejectBtn.textContent = 'Reject';
  rejectBtn.addEventListener('click', async () => {
    const comment = textarea.value.trim();
    if (!comment) {
      validation.classList.add('visible');
      textarea.focus();
      return;
    }
    validation.classList.remove('visible');
    approveBtn.disabled = true; rejectBtn.disabled = true;
    try {
      const res = await fetch('/api/modules/' + modPath + '/changes/' + filename + '/reject' + wtParam, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ comment }),
      });
      if (!res.ok) throw new Error('Reject failed: ' + res.status);
      reviewDraftChanges[index] = Object.assign({}, change, { review_comment: comment });
      renderReviewList();
    } catch (err) {
      showToast(err.message);
      approveBtn.disabled = false; rejectBtn.disabled = false;
    }
  });

  actions.append(approveBtn, rejectBtn);
  card.appendChild(actions);
  changesEl.appendChild(card);
}

function isOpArray(val) {
  return Array.isArray(val) && val.length > 0 && val.every(el => el !== null && typeof el === 'object' && 'op' in el);
}

function renderOpBlock(container, op) {
  function addLabeledPre(labelText, bodyText) {
    const lbl = document.createElement('div');
    lbl.className = 'review-op-label';
    lbl.textContent = labelText;
    container.appendChild(lbl);
    const pre = document.createElement('pre');
    pre.className = 'review-field-text';
    pre.textContent = bodyText != null ? String(bodyText) : '';
    container.appendChild(pre);
  }
  switch (op.op) {
    case 'append':  addLabeledPre('Append:', op.text); break;
    case 'prepend': addLabeledPre('Prepend:', op.text); break;
    case 'replace': addLabeledPre('Replace:', op.old); addLabeledPre('With:', op.new); break;
    case 'delete':  addLabeledPre('Delete:', op.text); break;
    default:        renderKeyValuePairs(container, op);
  }
}

function renderKeyValuePairs(container, obj) {
  const div = document.createElement('div');
  div.className = 'review-field-value';
  div.textContent = Object.entries(obj).map(([k, v]) =>
    k + ': ' + (v !== null && typeof v === 'object' ? JSON.stringify(v) : String(v))
  ).join('\n');
  container.appendChild(div);
}

function renderChangeFields(container, changes) {
  const fieldOrder = ['title', 'owner', 'status', 'deprecation_reason', 'description', 'behavior', 'technology', 'dependencies', 'acceptance_criteria'];
  const seen = new Set();

  for (const key of [...fieldOrder, ...Object.keys(changes)]) {
    if (seen.has(key) || !(key in changes)) continue;
    seen.add(key);
    const val = changes[key];
    if (val === null || val === undefined || val === '') continue;

    const field = document.createElement('div');
    field.className = 'review-field';

    const label = document.createElement('div');
    label.className = 'review-field-label';
    label.textContent = key.replace(/_/g, ' ');
    field.appendChild(label);

    if (key === 'acceptance_criteria' && Array.isArray(val)) {
      const list = document.createElement('ul');
      list.className = 'review-ac-list';
      for (const ac of val) {
        const item = document.createElement('li');
        item.className = 'review-ac-item';
        const idEl = document.createElement('span');
        idEl.className = 'review-ac-id';
        idEl.textContent = ac.id || '';
        const desc = document.createElement('span');
        desc.className = 'review-ac-desc';
        desc.textContent = ac.description || '';
        item.append(idEl, desc);
        list.appendChild(item);
      }
      field.appendChild(list);
    } else if (key === 'technology' && typeof val === 'object' && !Array.isArray(val)) {
      const table = document.createElement('table');
      table.className = 'review-tech-table';
      for (const [k, v] of Object.entries(val)) {
        const row = document.createElement('tr');
        const kCell = document.createElement('td'); kCell.textContent = k;
        const vCell = document.createElement('td'); vCell.textContent = String(v);
        row.append(kCell, vCell);
        table.appendChild(row);
      }
      field.appendChild(table);
    } else if (key === 'dependencies' && Array.isArray(val)) {
      const list = document.createElement('ul');
      list.style.paddingLeft = '18px';
      list.style.fontSize = '13px';
      for (const dep of val) {
        const li = document.createElement('li'); li.textContent = dep;
        list.appendChild(li);
      }
      field.appendChild(list);
    } else if (typeof val === 'string' && (key === 'description' || key === 'behavior' || val.includes('\n'))) {
      const pre = document.createElement('pre');
      pre.className = 'review-field-text';
      pre.textContent = val;
      field.appendChild(pre);
    } else if (isOpArray(val)) {
      for (const op of val) renderOpBlock(field, op);
    } else if (Array.isArray(val)) {
      const pre = document.createElement('pre');
      pre.className = 'review-field-text';
      pre.textContent = val.map(item =>
        (item !== null && typeof item === 'object')
          ? Object.entries(item).map(([k, v]) => '  ' + k + ': ' + (v != null ? String(v) : '')).join('\n')
          : '- ' + String(item)
      ).join('\n');
      field.appendChild(pre);
    } else if (typeof val === 'object' && val !== null) {
      renderKeyValuePairs(field, val);
    } else {
      const valueEl = document.createElement('div');
      valueEl.className = 'review-field-value';
      valueEl.textContent = String(val);
      field.appendChild(valueEl);
    }

    container.appendChild(field);
  }
}

function hideReviewPanel() {
  const panel = document.getElementById('review-panel');
  if (panel) panel.style.display = 'none';
  reviewDetailIndex = null;
  stopReviewPoller();
  schedulePoll();
}

// ── Keyboard shortcuts for review navigation ──────────────────────────────────

document.addEventListener('keydown', e => {
  if (reviewDetailIndex === null) return;
  const active = document.activeElement;
  if (active && (active.tagName === 'TEXTAREA' || active.tagName === 'INPUT')) return;
  if (e.key === 'j' || e.key === 'n') {
    if (reviewDetailIndex < reviewDraftChanges.length - 1) {
      e.preventDefault();
      renderReviewDetail(reviewDetailIndex + 1);
    }
  } else if (e.key === 'k' || e.key === 'p') {
    if (reviewDetailIndex > 0) {
      e.preventDefault();
      renderReviewDetail(reviewDetailIndex - 1);
    }
  }
});

// ── Resize handles ────────────────────────────────────────────────────────────

function initResizeHandles() {
  // Restore saved sidebar width.
  const savedLeftWidth = localStorage.getItem('eigen-left-width');
  if (savedLeftWidth) {
    document.documentElement.style.setProperty('--left-width', savedLeftWidth + 'px');
  }

  // Left sidebar handle (appended to #left, positioned on right edge via CSS).
  const leftEl = document.getElementById('left');
  if (leftEl) {
    const handle = document.createElement('div');
    handle.className = 'resize-handle';
    leftEl.appendChild(handle);

    handle.addEventListener('pointerdown', e => {
      e.preventDefault();
      handle.classList.add('dragging');
      const startX = e.clientX;
      const startWidth = leftEl.getBoundingClientRect().width;

      const overlay = document.createElement('div');
      overlay.className = 'resize-overlay';
      document.body.appendChild(overlay);

      const clamp = x => Math.min(Math.max(x, 180), window.innerWidth * 0.5);

      const onMove = ev => {
        document.documentElement.style.setProperty('--left-width', clamp(startWidth + ev.clientX - startX) + 'px');
      };
      const onUp = ev => {
        const w = clamp(startWidth + ev.clientX - startX);
        document.documentElement.style.setProperty('--left-width', w + 'px');
        localStorage.setItem('eigen-left-width', Math.round(w));
        handle.classList.remove('dragging');
        overlay.remove();
        document.removeEventListener('pointermove', onMove);
        document.removeEventListener('pointerup', onUp);
      };
      document.addEventListener('pointermove', onMove);
      document.addEventListener('pointerup', onUp);
    });
  }

  // Review panel handle (prepended to #review-panel, positioned on left edge via CSS).
  const panelEl = document.getElementById('review-panel');
  if (panelEl) {
    const handle = document.createElement('div');
    handle.className = 'resize-handle';
    panelEl.prepend(handle);

    handle.addEventListener('pointerdown', e => {
      e.preventDefault();
      handle.classList.add('dragging');
      const startX = e.clientX;
      const startWidth = panelEl.getBoundingClientRect().width;

      const overlay = document.createElement('div');
      overlay.className = 'resize-overlay';
      document.body.appendChild(overlay);

      const clamp = x => Math.min(Math.max(x, 320), window.innerWidth * 0.8);

      const onMove = ev => {
        panelEl.style.width = clamp(startWidth - (ev.clientX - startX)) + 'px';
      };
      const onUp = ev => {
        const w = clamp(startWidth - (ev.clientX - startX));
        panelEl.style.width = w + 'px';
        localStorage.setItem('eigen-review-width', Math.round(w));
        handle.classList.remove('dragging');
        overlay.remove();
        document.removeEventListener('pointermove', onMove);
        document.removeEventListener('pointerup', onUp);
      };
      document.addEventListener('pointermove', onMove);
      document.addEventListener('pointerup', onUp);
    });
  }
}

// ── Worktree Poller ───────────────────────────────────────────────────────────

let worktreePollerTimer = null;

function startWorktreePoller() {
  stopWorktreePoller();
  scheduleWorktreePoll();
}

function stopWorktreePoller() {
  if (worktreePollerTimer !== null) {
    clearTimeout(worktreePollerTimer);
    worktreePollerTimer = null;
  }
}

function scheduleWorktreePoll() {
  worktreePollerTimer = setTimeout(pollWorktrees, 3000);
}

async function pollWorktrees() {
  try {
    const res = await fetch('/api/worktrees');
    if (!res.ok) { scheduleWorktreePoll(); return; }
    const newWorktrees = await res.json();
    // Check if the worktree set has changed by comparing names.
    const oldNames = allWorktrees.map(w => w.name).sort().join(',');
    const newNames = newWorktrees.map(w => w.name).sort().join(',');
    if (oldNames !== newNames) {
      allWorktrees = newWorktrees;
      // Also refresh modules.
      const modRes = await fetch('/api/modules');
      if (modRes.ok) allModules = await modRes.json();
      renderTreeView();
      updatePageTitle();
    }
  } catch (_) {
    // ignore
  }
  scheduleWorktreePoll();
}

// ── Start ─────────────────────────────────────────────────────────────────────

init();
