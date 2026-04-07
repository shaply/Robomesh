<script>
  /**
   * Test Robot Handler Page — Full network diagnostics dashboard.
   *
   * @prop {object} robot - Robot data
   * @prop {function} fetchBackend - Authenticated API call helper
   * @prop {function} sendToHandler - Send a message to the handler process
   */
  let { robot = {}, fetchBackend = null, sendToHandler = null } = $props();

  // --- State ---
  let signalDbm = $state(-45);
  let ssid = $state('RoboNet-5G');
  let frequency = $state('5 GHz');
  let linkSpeed = $state(866);
  let testsRun = $state(0);

  let speedRunning = $state(false);
  let speedResult = $state(null);
  let speedHistory = $state([]);

  let pingTarget = $state('8.8.8.8');
  let pingCount = $state(4);
  let pingRunning = $state(false);
  let pingResult = $state(null);

  let logs = $state([]);

  // --- Derived ---
  function signalPercent(dbm) {
    return Math.max(0, Math.min(100, ((dbm + 90) / 70) * 100));
  }

  function signalLabel(dbm) {
    if (dbm >= -30) return 'Excellent';
    if (dbm >= -50) return 'Good';
    if (dbm >= -60) return 'Fair';
    if (dbm >= -70) return 'Weak';
    return 'Poor';
  }

  function signalColor(dbm) {
    if (dbm >= -50) return 'var(--success, #22c55e)';
    if (dbm >= -60) return 'var(--warning, #f59e0b)';
    return 'var(--error, #ef4444)';
  }

  let pct = $derived(signalPercent(signalDbm));
  let qualityLabel = $derived(signalLabel(signalDbm));
  let qualityColor = $derived(signalColor(signalDbm));

  // --- Actions ---
  function addLog(type, text) {
    logs = [...logs, { type, text, time: new Date().toLocaleTimeString() }];
    if (logs.length > 100) logs = logs.slice(-100);
  }

  async function runSpeedTest() {
    if (!sendToHandler || speedRunning) return;
    speedRunning = true;
    addLog('sent', 'Starting speed test...');
    sendToHandler(JSON.stringify({ command: 'speed_test' }));

    // Simulate receiving the result (in production, SSE or polling would update this)
    setTimeout(() => {
      // Generate mock result client-side as fallback
      const dl = (100 + Math.random() * 100).toFixed(2);
      const ul = (30 + Math.random() * 40).toFixed(2);
      const lat = (5 + Math.random() * 30).toFixed(1);
      const jit = (1 + Math.random() * 14).toFixed(1);

      const result = {
        download_mbps: parseFloat(dl),
        upload_mbps: parseFloat(ul),
        latency_ms: parseFloat(lat),
        jitter_ms: parseFloat(jit),
        timestamp: Math.floor(Date.now() / 1000),
      };

      speedResult = result;
      speedHistory = [...speedHistory, result];
      if (speedHistory.length > 20) speedHistory = speedHistory.slice(-20);
      testsRun++;
      speedRunning = false;
      addLog('recv', `Speed test complete: ${result.download_mbps} Mbps down, ${result.upload_mbps} Mbps up`);
    }, 1500);
  }

  async function runPing() {
    if (!sendToHandler || pingRunning) return;
    pingRunning = true;
    addLog('sent', `Pinging ${pingTarget} (${pingCount} packets)...`);
    sendToHandler(JSON.stringify({ command: 'ping', target: pingTarget, count: pingCount }));

    setTimeout(() => {
      const results = Array.from({ length: pingCount }, () => parseFloat((5 + Math.random() * 25).toFixed(2)));
      const avg = (results.reduce((a, b) => a + b, 0) / results.length).toFixed(2);

      pingResult = {
        target: pingTarget,
        count: pingCount,
        results_ms: results,
        avg_ms: parseFloat(avg),
        min_ms: Math.min(...results),
        max_ms: Math.max(...results),
        packet_loss: 0,
      };
      pingRunning = false;
      addLog('recv', `Ping complete: avg ${avg} ms, 0% loss`);
    }, 1000);
  }

  function requestStatus() {
    if (!sendToHandler) return;
    addLog('sent', 'Requesting status...');
    sendToHandler(JSON.stringify({ command: 'status' }));

    setTimeout(() => {
      signalDbm = Math.max(-90, Math.min(-20, signalDbm + (Math.random() > 0.5 ? 3 : -3)));
      addLog('recv', `Signal: ${signalDbm} dBm (${signalLabel(signalDbm)}), SSID: ${ssid}`);
    }, 300);
  }

  function formatTime(ts) {
    return new Date(ts * 1000).toLocaleTimeString();
  }
</script>

<div class="handler-page">
  <!-- Signal Overview -->
  <section class="section signal-overview">
    <h2>Network Status</h2>
    <div class="signal-grid">
      <div class="signal-gauge">
        <svg viewBox="0 0 120 70" class="gauge-svg">
          <path d="M 10 65 A 50 50 0 0 1 110 65" fill="none" stroke="var(--bg-hover, #243352)" stroke-width="8" stroke-linecap="round" />
          <path d="M 10 65 A 50 50 0 0 1 110 65" fill="none" stroke={qualityColor} stroke-width="8" stroke-linecap="round"
                stroke-dasharray={`${pct * 1.57} 157`} />
          <text x="60" y="52" text-anchor="middle" fill={qualityColor} font-size="16" font-weight="700" font-family="monospace">{signalDbm}</text>
          <text x="60" y="64" text-anchor="middle" fill="var(--text-muted, #5a6d85)" font-size="8">dBm</text>
        </svg>
        <span class="quality-label" style="color: {qualityColor};">{qualityLabel}</span>
      </div>
      <div class="info-list">
        <div class="info-row">
          <span class="info-key">SSID</span>
          <span class="info-val">{ssid}</span>
        </div>
        <div class="info-row">
          <span class="info-key">Frequency</span>
          <span class="info-val">{frequency}</span>
        </div>
        <div class="info-row">
          <span class="info-key">Link Speed</span>
          <span class="info-val">{linkSpeed} Mbps</span>
        </div>
        <div class="info-row">
          <span class="info-key">Tests Run</span>
          <span class="info-val">{testsRun}</span>
        </div>
        <div class="info-row">
          <span class="info-key">Robot IP</span>
          <span class="info-val mono">{robot.ip || 'N/A'}</span>
        </div>
      </div>
    </div>
    <button class="btn btn-secondary" onclick={requestStatus}>Refresh Status</button>
  </section>

  <!-- Speed Test -->
  <section class="section">
    <h2>Speed Test</h2>
    <button class="btn btn-primary" onclick={runSpeedTest} disabled={speedRunning}>
      {speedRunning ? 'Running...' : 'Run Speed Test'}
    </button>

    {#if speedResult}
      <div class="result-grid">
        <div class="result-card download">
          <span class="result-val">{speedResult.download_mbps}</span>
          <span class="result-label">Download Mbps</span>
        </div>
        <div class="result-card upload">
          <span class="result-val">{speedResult.upload_mbps}</span>
          <span class="result-label">Upload Mbps</span>
        </div>
        <div class="result-card latency">
          <span class="result-val">{speedResult.latency_ms}</span>
          <span class="result-label">Latency ms</span>
        </div>
        <div class="result-card jitter">
          <span class="result-val">{speedResult.jitter_ms}</span>
          <span class="result-label">Jitter ms</span>
        </div>
      </div>
    {/if}

    {#if speedHistory.length > 1}
      <div class="history-section">
        <h3>History ({speedHistory.length} tests)</h3>
        <div class="history-table-wrap">
          <table class="history-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Down</th>
                <th>Up</th>
                <th>Latency</th>
                <th>Jitter</th>
              </tr>
            </thead>
            <tbody>
              {#each [...speedHistory].reverse() as entry}
                <tr>
                  <td class="mono">{formatTime(entry.timestamp)}</td>
                  <td>{entry.download_mbps}</td>
                  <td>{entry.upload_mbps}</td>
                  <td>{entry.latency_ms}</td>
                  <td>{entry.jitter_ms}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  </section>

  <!-- Ping -->
  <section class="section">
    <h2>Ping</h2>
    <div class="ping-controls">
      <input type="text" class="input" bind:value={pingTarget} placeholder="Target host" />
      <input type="number" class="input input-sm" bind:value={pingCount} min="1" max="20" />
      <button class="btn btn-primary" onclick={runPing} disabled={pingRunning}>
        {pingRunning ? 'Pinging...' : 'Ping'}
      </button>
    </div>

    {#if pingResult}
      <div class="ping-result">
        <div class="ping-stat">
          <span class="ping-stat-val">{pingResult.avg_ms}</span>
          <span class="ping-stat-label">avg ms</span>
        </div>
        <div class="ping-stat">
          <span class="ping-stat-val">{pingResult.min_ms}</span>
          <span class="ping-stat-label">min ms</span>
        </div>
        <div class="ping-stat">
          <span class="ping-stat-val">{pingResult.max_ms}</span>
          <span class="ping-stat-label">max ms</span>
        </div>
        <div class="ping-stat">
          <span class="ping-stat-val">{pingResult.packet_loss}%</span>
          <span class="ping-stat-label">loss</span>
        </div>
      </div>
      <div class="ping-detail mono">
        {#each pingResult.results_ms as rtt, i}
          <span class="ping-rtt">#{i + 1}: {rtt} ms</span>
        {/each}
      </div>
    {/if}
  </section>

  <!-- Activity Log -->
  <section class="section">
    <h2>Activity Log</h2>
    <div class="log-container">
      {#each [...logs].reverse() as log}
        <div class="log-line">
          <span class="log-time">{log.time}</span>
          <span class="log-dir" class:log-sent={log.type === 'sent'} class:log-recv={log.type === 'recv'}>
            {log.type === 'sent' ? '>' : '<'}
          </span>
          <span class="log-text">{log.text}</span>
        </div>
      {/each}
      {#if logs.length === 0}
        <div class="log-empty">No activity yet. Run a speed test or ping to get started.</div>
      {/if}
    </div>
  </section>
</div>

<style>
  .handler-page {
    padding: 1.5rem;
    color: var(--text-primary, #e8edf5);
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    max-width: 900px;
  }

  .section {
    background: var(--bg-elevated, #1c2740);
    border: 1px solid var(--border, #2a3a54);
    border-radius: var(--radius, 10px);
    padding: 1.25rem;
  }

  h2 {
    margin: 0 0 1rem;
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary, #e8edf5);
  }

  h3 {
    margin: 1rem 0 0.5rem;
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-secondary, #8899b0);
  }

  /* Signal Overview */
  .signal-grid {
    display: flex;
    gap: 1.5rem;
    align-items: center;
    margin-bottom: 1rem;
  }

  .signal-gauge {
    display: flex;
    flex-direction: column;
    align-items: center;
    min-width: 140px;
  }

  .gauge-svg {
    width: 140px;
    height: 80px;
  }

  .quality-label {
    font-size: 0.8rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-top: 0.25rem;
  }

  .info-list {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .info-row {
    display: flex;
    justify-content: space-between;
    font-size: 0.85rem;
  }

  .info-key {
    color: var(--text-muted, #5a6d85);
    font-weight: 500;
  }

  .info-val {
    color: var(--text-secondary, #8899b0);
  }

  .mono {
    font-family: var(--font-mono, monospace);
    font-size: 0.8rem;
  }

  /* Buttons */
  .btn {
    padding: 0.5rem 1.25rem;
    border: none;
    border-radius: var(--radius-sm, 6px);
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s, opacity 0.15s;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    background: var(--accent, #38bdf8);
    color: #0b1120;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--accent-hover, #0ea5e9);
  }

  .btn-secondary {
    background: var(--bg-hover, #243352);
    color: var(--text-secondary, #8899b0);
    border: 1px solid var(--border, #2a3a54);
  }

  .btn-secondary:hover:not(:disabled) {
    background: var(--border, #2a3a54);
    color: var(--text-primary, #e8edf5);
  }

  /* Speed Test Results */
  .result-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.75rem;
    margin-top: 1rem;
  }

  .result-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 0.75rem;
    border-radius: var(--radius-sm, 6px);
    background: var(--bg-hover, #243352);
  }

  .result-val {
    font-size: 1.4rem;
    font-weight: 700;
    font-family: var(--font-mono, monospace);
  }

  .result-label {
    font-size: 0.7rem;
    color: var(--text-muted, #5a6d85);
    text-transform: uppercase;
    margin-top: 0.2rem;
  }

  .download .result-val { color: var(--success, #22c55e); }
  .upload .result-val { color: var(--accent, #38bdf8); }
  .latency .result-val { color: var(--warning, #f59e0b); }
  .jitter .result-val { color: var(--text-secondary, #8899b0); }

  /* History Table */
  .history-table-wrap {
    overflow-x: auto;
  }

  .history-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.8rem;
  }

  .history-table th {
    text-align: left;
    padding: 0.4rem 0.75rem;
    color: var(--text-muted, #5a6d85);
    font-weight: 600;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    border-bottom: 1px solid var(--border, #2a3a54);
  }

  .history-table td {
    padding: 0.35rem 0.75rem;
    color: var(--text-secondary, #8899b0);
    border-bottom: 1px solid var(--border-light, #1e2d45);
    font-family: var(--font-mono, monospace);
  }

  .history-table tbody tr:hover {
    background: var(--bg-hover, #243352);
  }

  /* Ping */
  .ping-controls {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .input {
    padding: 0.5rem 0.75rem;
    border-radius: var(--radius-sm, 6px);
    border: 1px solid var(--border, #2a3a54);
    background: var(--bg-base, #0b1120);
    color: var(--text-primary, #e8edf5);
    font-size: 0.85rem;
    font-family: var(--font-mono, monospace);
  }

  .input:focus {
    outline: none;
    border-color: var(--accent, #38bdf8);
  }

  .input-sm {
    width: 70px;
  }

  .ping-result {
    display: flex;
    gap: 1.5rem;
    margin-top: 1rem;
    padding: 0.75rem;
    background: var(--bg-hover, #243352);
    border-radius: var(--radius-sm, 6px);
  }

  .ping-stat {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .ping-stat-val {
    font-size: 1.1rem;
    font-weight: 700;
    font-family: var(--font-mono, monospace);
    color: var(--accent, #38bdf8);
  }

  .ping-stat-label {
    font-size: 0.65rem;
    color: var(--text-muted, #5a6d85);
    text-transform: uppercase;
  }

  .ping-detail {
    display: flex;
    gap: 1rem;
    margin-top: 0.5rem;
    flex-wrap: wrap;
  }

  .ping-rtt {
    font-size: 0.75rem;
    color: var(--text-secondary, #8899b0);
  }

  /* Activity Log */
  .log-container {
    max-height: 250px;
    overflow-y: auto;
    font-family: var(--font-mono, monospace);
    font-size: 0.8rem;
    background: var(--bg-base, #0b1120);
    border-radius: var(--radius-sm, 6px);
    padding: 0.5rem;
  }

  .log-line {
    display: flex;
    gap: 0.5rem;
    padding: 0.2rem 0;
    align-items: baseline;
  }

  .log-time {
    color: var(--text-muted, #5a6d85);
    font-size: 0.7rem;
    flex-shrink: 0;
  }

  .log-dir {
    flex-shrink: 0;
    font-weight: 700;
    width: 1rem;
    text-align: center;
  }

  .log-sent { color: var(--accent, #38bdf8); }
  .log-recv { color: var(--success, #22c55e); }

  .log-text {
    color: var(--text-secondary, #8899b0);
  }

  .log-empty {
    color: var(--text-muted, #5a6d85);
    text-align: center;
    padding: 1rem;
    font-style: italic;
  }

  @media (max-width: 600px) {
    .signal-grid {
      flex-direction: column;
    }
    .result-grid {
      grid-template-columns: repeat(2, 1fr);
    }
    .ping-controls {
      flex-wrap: wrap;
    }
  }
</style>
