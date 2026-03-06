// ==================== Constants ====================
const API_URL = window.location.origin + "/run";
const HISTORY_KEY = "parallelhttp_history";
const MAX_HISTORY = 50;

// ==================== State ====================
let tabs = [];
let activeTabId = null;
let tabCounter = 0;

// ==================== Tab Helpers ====================

function createDefaultFormState() {
    return {
        method: 'GET',
        endpoint: '',
        body: '',
        headers: '',
        request_timeout: 10000,
        max_duration: 30000,
        parallel: 5,
    };
}

function createTab(name) {
    tabCounter++;
    const id = `tab-${tabCounter}`;
    const tab = {
        id,
        name: name || `Tab ${tabCounter}`,
        form: createDefaultFormState(),
        result: null,
        running: false,
        error: null,
    };
    tabs.push(tab);
    return tab;
}

function getTab(id) {
    return tabs.find(t => t.id === id);
}

function closeTab(id) {
    if (tabs.length <= 1) return;
    const idx = tabs.findIndex(t => t.id === id);
    if (idx === -1) return;
    tabs.splice(idx, 1);
    if (activeTabId === id) {
        switchTab(tabs[Math.min(idx, tabs.length - 1)].id, true);
    } else {
        renderTabBar();
    }
}

// ==================== Form State ====================

function readFormFromDOM() {
    return {
        method: document.getElementById('method').value,
        endpoint: document.getElementById('endpoint').value,
        body: document.getElementById('body').value,
        headers: document.getElementById('headers').value,
        request_timeout: parseInt(document.getElementById('request_timeout').value, 10) || 0,
        max_duration: parseInt(document.getElementById('max_duration').value, 10) || 0,
        parallel: parseInt(document.getElementById('parallel').value, 10) || 1,
    };
}

function writeFormToDOM(form) {
    document.getElementById('method').value = form.method;
    document.getElementById('endpoint').value = form.endpoint;
    document.getElementById('body').value = form.body;
    document.getElementById('headers').value = form.headers;
    document.getElementById('request_timeout').value = form.request_timeout;
    document.getElementById('max_duration').value = form.max_duration;
    document.getElementById('parallel').value = form.parallel;
}

// ==================== Tab Switching ====================

function switchTab(newId, skipSave) {
    if (!skipSave && activeTabId) {
        const current = getTab(activeTabId);
        if (current) current.form = readFormFromDOM();
    }
    activeTabId = newId;
    const tab = getTab(newId);
    if (!tab) return;
    writeFormToDOM(tab.form);
    renderTabBar();
    renderOutput(tab);
}

// ==================== Tab Bar ====================

function renderTabBar() {
    const bar = document.getElementById('requestTabBar');
    const addBtn = document.getElementById('addTab');
    bar.querySelectorAll('.request-tab').forEach(el => el.remove());

    tabs.forEach(tab => {
        const el = document.createElement('div');
        el.className = `request-tab d-flex align-items-center gap-2 px-3 py-2${tab.id === activeTabId ? ' request-tab-active' : ''}`;
        el.dataset.tabId = tab.id;

        const label = document.createElement('span');
        label.className = 'tab-label';
        label.textContent = tab.name;
        el.appendChild(label);

        if (tab.running) {
            const spinner = document.createElement('span');
            spinner.className = 'spinner-border spinner-border-sm text-primary';
            spinner.setAttribute('role', 'status');
            el.appendChild(spinner);
        }

        if (tabs.length > 1) {
            const closeBtn = document.createElement('button');
            closeBtn.type = 'button';
            closeBtn.className = 'tab-close-btn';
            closeBtn.innerHTML = '&times;';
            closeBtn.title = 'Close tab';
            closeBtn.addEventListener('click', e => { e.stopPropagation(); closeTab(tab.id); });
            el.appendChild(closeBtn);
        }

        el.addEventListener('click', () => switchTab(tab.id));
        bar.insertBefore(el, addBtn);
    });
}

// ==================== Output ====================

function renderOutput(tab) {
    const output = document.getElementById('output');
    const statusEl = document.getElementById('status');
    const downloadBtn = document.getElementById('downloadCsv');
    const runBtn = document.getElementById('run');

    runBtn.disabled = tab.running;

    if (tab.running) {
        statusEl.textContent = 'Running...';
        output.classList.add('d-none');
        downloadBtn.classList.add('d-none');
        return;
    }

    if (tab.error) {
        statusEl.textContent = 'Error: ' + tab.error;
        output.classList.add('d-none');
        downloadBtn.classList.add('d-none');
        return;
    }

    if (tab.result) {
        statusEl.textContent = '';
        renderResults(tab.result);
        output.classList.remove('d-none');
        downloadBtn.classList.remove('d-none');
        downloadBtn.onclick = () => downloadCSV(tab.result.results || []);
        return;
    }

    statusEl.textContent = '';
    output.classList.add('d-none');
    downloadBtn.classList.add('d-none');
}

function renderResults(json) {
    const results = json.results || [];
    const summary = json.summary || {};

    const cards = [
        ['Total', summary.total_requests ?? results.length],
        ['Success', summary.success_count ?? 0],
        ['Errors', summary.error_count ?? 0],
        ['Avg Duration', summary.avg_duration ?? '-'],
    ];

    document.getElementById('summary').innerHTML = cards.map(([title, val]) => `
        <div class="col">
            <div class="card p-3 text-center shadow-sm">
                <div class="small">Requests</div>
                <div class="text-muted small">${title}</div>
                <div class="fs-4 fw-bold mt-1">${val}</div>
            </div>
        </div>
    `).join('');

    const latencyCards = [
        ['P50', summary.latency?.p50 ?? 0],
        ['P90', summary.latency?.p90 ?? 0],
        ['P99', summary.latency?.p99 ?? 0],
    ];

    document.getElementById('latency').innerHTML = latencyCards.map(([title, val]) => `
        <div class="col">
            <div class="card p-4 text-center shadow-sm">
                <div class="small">Latency</div>
                <div class="text-muted small">${title}</div>
                <div class="fs-4 fw-bold mt-1">${val}</div>
            </div>
        </div>
    `).join('');

    let table = `
    <table class="table">
        <thead class="thead-dark">
            <tr>
                <th>#</th>
                <th>Time</th>
                <th>Status Code</th>
                <th>Duration</th>
                <th>Error</th>
            </tr>
        </thead>
        <tbody>`;

    results.forEach((r, i) => {
        table += `
            <tr>
                <td>${i + 1}</td>
                <td>${r.time || '-'}</td>
                <td>${r.status_code || '-'}</td>
                <td>${r.duration || '-'}</td>
                <td class="text-danger">${r.error || ''}</td>
            </tr>`;
    });

    table += '</tbody></table>';
    document.getElementById('resultsArea').innerHTML = table;
    document.getElementById('raw').textContent = JSON.stringify(json, null, 2);
}

// ==================== Run ====================

async function runCurrentTab() {
    const tab = getTab(activeTabId);
    if (!tab || tab.running) return;

    tab.form = readFormFromDOM();
    tab.running = true;
    tab.error = null;
    tab.result = null;

    renderTabBar();
    renderOutput(tab);

    const payload = {
        method: tab.form.method,
        endpoint: tab.form.endpoint.trim(),
        parallel: tab.form.parallel || 1,
        request_timeout: tab.form.request_timeout || 0,
        max_duration: tab.form.max_duration || 0,
    };

    const bodyText = tab.form.body.trim();
    if (bodyText) {
        try { payload.body = JSON.parse(bodyText); } catch { payload.body = bodyText; }
    }

    const headersText = tab.form.headers.trim();
    if (headersText) {
        try { payload.headers = JSON.parse(headersText); } catch { payload.headers = headersText; }
    }

    try {
        const res = await fetch(API_URL, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        const json = await res.json();
        tab.result = json;
        tab.error = null;
        saveToHistory(tab.name, payload, json);
    } catch (err) {
        tab.error = err.message;
    } finally {
        tab.running = false;
        renderTabBar();
        if (activeTabId === tab.id) {
            renderOutput(tab);
        }
    }
}

// ==================== History ====================

function loadHistory() {
    try {
        return JSON.parse(localStorage.getItem(HISTORY_KEY)) || [];
    } catch {
        return [];
    }
}

function saveToHistory(tabName, payload, result) {
    const history = loadHistory();
    history.unshift({
        id: Date.now(),
        timestamp: new Date().toISOString(),
        tabName,
        payload,
        result,
    });
    if (history.length > MAX_HISTORY) history.length = MAX_HISTORY;
    localStorage.setItem(HISTORY_KEY, JSON.stringify(history));
    renderHistory();
}

function renderHistory() {
    const history = loadHistory();
    const list = document.getElementById('historyList');
    const empty = document.getElementById('historyEmpty');

    if (!history.length) {
        list.innerHTML = '';
        empty.classList.remove('d-none');
        return;
    }

    empty.classList.add('d-none');
    list.innerHTML = history.map(entry => {
        const s = entry.result?.summary || {};
        const ts = new Date(entry.timestamp).toLocaleString();
        const method = escapeHtml(entry.payload?.method || '-');
        const endpoint = escapeHtml(entry.payload?.endpoint || '-');
        const total = s.total_requests ?? '-';
        const success = s.success_count ?? '-';
        const errors = s.error_count ?? '-';
        return `
            <div class="card history-entry p-3">
                <div class="d-flex align-items-start gap-3">
                    <div class="flex-grow-1 overflow-hidden">
                        <div class="d-flex align-items-center flex-wrap gap-2 mb-1">
                            <span class="badge bg-primary">${method}</span>
                            <span class="text-truncate fw-semibold small" title="${endpoint}" style="max-width:320px">${endpoint}</span>
                            <span class="text-muted small ms-auto text-nowrap">${ts}</span>
                        </div>
                        <div class="small text-muted">
                            Total: <strong>${total}</strong>&ensp;
                            Success: <strong class="text-success">${success}</strong>&ensp;
                            Errors: <strong class="text-danger">${errors}</strong>
                        </div>
                    </div>
                    <button class="btn btn-sm btn-outline-secondary flex-shrink-0 restore-btn" data-history-id="${entry.id}">
                        Restore
                    </button>
                </div>
            </div>`;
    }).join('');

    list.querySelectorAll('.restore-btn').forEach(btn => {
        btn.addEventListener('click', () => restoreFromHistory(parseInt(btn.dataset.historyId, 10)));
    });
}

function restoreFromHistory(id) {
    const entry = loadHistory().find(e => e.id === id);
    if (!entry) return;
    const tab = getTab(activeTabId);
    if (!tab) return;
    const p = entry.payload || {};
    tab.form = {
        method: p.method || 'GET',
        endpoint: p.endpoint || '',
        body: p.body ? (typeof p.body === 'string' ? p.body : JSON.stringify(p.body, null, 2)) : '',
        headers: p.headers ? (typeof p.headers === 'string' ? p.headers : JSON.stringify(p.headers, null, 2)) : '',
        request_timeout: p.request_timeout || 10000,
        max_duration: p.max_duration || 30000,
        parallel: p.parallel || 5,
    };
    writeFormToDOM(tab.form);
}

// ==================== CSV ====================

function downloadCSV(results) {
    if (!results?.length) return alert('No results to download');
    const headers = ['#', 'Time', 'Status Code', 'Duration', 'Error'];
    const rows = results.map((r, i) =>
        [i + 1, r.time, r.status_code, r.duration, r.error || ''].join(',')
    );
    const csv = '\uFEFF' + headers.join(',') + '\n' + rows.join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const a = Object.assign(document.createElement('a'), {
        href: URL.createObjectURL(blob),
        download: 'parallel_results.csv',
    });
    a.click();
    URL.revokeObjectURL(a.href);
}

// ==================== Utils ====================

function escapeHtml(str) {
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

// ==================== Init ====================

document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-bs-toggle="tooltip"]').forEach(el => new bootstrap.Tooltip(el));

    const first = createTab('Tab 1');
    activeTabId = first.id;
    renderTabBar();
    renderHistory();

    document.getElementById('frm').addEventListener('submit', ev => {
        ev.preventDefault();
        runCurrentTab();
    });

    document.getElementById('addTab').addEventListener('click', () => {
        const t = createTab();
        switchTab(t.id);
    });

    document.getElementById('clearHistory').addEventListener('click', () => {
        if (!confirm('Clear all history?')) return;
        localStorage.removeItem(HISTORY_KEY);
        renderHistory();
    });
});
