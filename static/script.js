const form = document.getElementById('frm');
const runBtn = document.getElementById('run');
const downloadBtn = document.getElementById('downloadCsv');
const statusEl = document.getElementById('status');
const output = document.getElementById('output');
const raw = document.getElementById('raw');
const resultsArea = document.getElementById('resultsArea');
const summaryEl = document.getElementById('summary');
const latencyEl = document.getElementById('latency')
const url = "http://localhost:8080/run"

form.addEventListener('submit', async (ev) => {
    ev.preventDefault();
    runBtn.disabled = true;
    
    output.classList.add('d-none');
    statusEl.textContent = 'Running...';
    resultsArea.innerHTML = '';
    summaryEl.innerHTML = '';
    latencyEl.innerHTML = ''
    raw.textContent = '';
    downloadBtn.classList.add('d-none');

    const payload = {
        method: document.getElementById('method').value,
        endpoint: document.getElementById('endpoint').value.trim(),
        parallel: parseInt(document.getElementById('parallel').value, 10) || 1,
        request_timeout: parseInt(document.getElementById('request_timeout').value, 10) || 0,
        max_duration: parseInt(document.getElementById('max_duration').value, 10) || 0,
    };

    const bodyText = document.getElementById('body').value.trim();
    if (bodyText) {
        try { payload.body = JSON.parse(bodyText); } catch { payload.body = bodyText; }
    }

    const headersText = document.getElementById('headers').value.trim();
    if (headersText) {
        try { payload.headers = JSON.parse(headersText); } catch { payload.headers = headersText; }
    }

    try {
        const res = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json'},
            body: JSON.stringify(payload)
        });

        const json = await res.json();

        raw.textContent = JSON.stringify(json, null, 2);
        renderResults(json);

        output.classList.remove('d-none');
        statusEl.textContent = '';
        downloadBtn.classList.remove('d-none');

        downloadBtn.onclick = () => downloadCSV(json.results || []);

    } catch (err) {
        statusEl.textContent = 'Error: ' + err.message;
    } finally {
        runBtn.disabled = false;
    }
});

function renderResults(json) {
    const results = json.results || [];
    const summary = json.summary || {};

    const cards = [
        ['Total', summary.total_requests ?? results.length],
        ['Success', summary.success_count ?? 0],
        ['Errors', summary.error_count ?? 0],
        ['Avg Duration', summary.avg_duration ?? '-'],
    ];

    summaryEl.innerHTML = cards.map(([title, val]) => `
        <div class="col">
            <div class="card p-3 text-center shadow-sm">
                <div class="small">Requests</div>
                <div class="text-muted small">${title}</div>
                <div class="fs-4 fw-bold mt-1">${val}</div>
            </div>
        </div>
    `).join('');

    const latencyCars = [
        ['P50', summary.latency.p50 ?? 0],
        ['P90', summary.latency.p90 ?? 0],
        ['P99', summary.latency.p99 ?? 0],
    ];
   
    latencyEl.innerHTML = latencyCars.map(([title, val]) => `
        <div class="col">
            <div class="card p-4 text-center shadow-sm">
                <div class="small">Latency</div>
                <div class="text-muted small">${title}</div>
                <div class="fs-4 fw-bold mt-1">${val}</div>
            </div>
        </div>
    `).join('');

    // Bootstrap table
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
            </tr>
        `;
    });

    table += '</tbody></table>';
    resultsArea.innerHTML = table;
}

function downloadCSV(results) {
    if (!results?.length) return alert('No results to download');

    const headers = ['#','Time','Status Code','Duration','Error'];
    const rows = results.map((r, i) =>
        [i + 1, r.time, r.status_code, r.duration, r.error || ''].join(',')
    );

    const csvContent = '\uFEFF' + headers.join(',') + '\n' + rows.join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);

    const a = document.createElement('a');
    a.href = url;
    a.download = 'parallelhttp_results.csv';
    a.click();

    URL.revokeObjectURL(url);
}

// Handle tooltips
 document.addEventListener('DOMContentLoaded', function () {
  const tooltipTriggerList = [].slice.call(
    document.querySelectorAll('[data-bs-toggle="tooltip"]')
  );
  tooltipTriggerList.forEach(el => new bootstrap.Tooltip(el));
});