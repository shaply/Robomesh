<script>
  /**
   * Test Robot Card — Network diagnostics at a glance.
   * Shows wifi signal strength, last speed test, and connection quality.
   *
   * @prop {object} robot - Robot data (device_id, ip, robot_type, status, etc.)
   */
  let { robot = {} } = $props();

  // Simulated live metrics (in a real setup these would come from SSE/events)
  let signalDbm = $state(-45);
  let lastDownload = $state(null);
  let lastUpload = $state(null);
  let lastLatency = $state(null);

  function signalPercent(dbm) {
    // Map -90..-20 dBm to 0..100%
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
  let label = $derived(signalLabel(signalDbm));
  let color = $derived(signalColor(signalDbm));
</script>

<div class="test-card">
  <div class="card-header">
    <span class="type-icon">&#x1F4F6;</span>
    <div class="header-text">
      <span class="type-label">Network Diagnostics</span>
      <span class="device-id">{robot.device_id || robot.uuid || 'N/A'}</span>
    </div>
  </div>

  <div class="signal-section">
    <div class="signal-bar-track">
      <div class="signal-bar-fill" style="width: {pct}%; background: {color};"></div>
    </div>
    <div class="signal-meta">
      <span class="signal-label" style="color: {color};">{label}</span>
      <span class="signal-dbm">{signalDbm} dBm</span>
    </div>
  </div>

  <div class="metrics">
    {#if lastDownload !== null}
      <div class="metric">
        <span class="metric-val">{lastDownload}</span>
        <span class="metric-unit">Mbps &darr;</span>
      </div>
      <div class="metric">
        <span class="metric-val">{lastUpload}</span>
        <span class="metric-unit">Mbps &uarr;</span>
      </div>
      <div class="metric">
        <span class="metric-val">{lastLatency}</span>
        <span class="metric-unit">ms</span>
      </div>
    {:else}
      <div class="no-data">No speed test yet</div>
    {/if}
  </div>
</div>

<style>
  .test-card {
    padding: 1rem;
    border-radius: 0.5rem;
    background: var(--bg-elevated, #1c2740);
    color: var(--text-primary, #e8edf5);
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .card-header {
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }

  .type-icon {
    font-size: 1.4rem;
    line-height: 1;
  }

  .header-text {
    display: flex;
    flex-direction: column;
  }

  .type-label {
    font-weight: 600;
    font-size: 0.9rem;
    color: var(--text-primary, #e8edf5);
  }

  .device-id {
    font-family: monospace;
    font-size: 0.7rem;
    color: var(--text-muted, #5a6d85);
  }

  .signal-section {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .signal-bar-track {
    height: 6px;
    border-radius: 3px;
    background: var(--bg-hover, #243352);
    overflow: hidden;
  }

  .signal-bar-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.4s ease, background 0.4s ease;
  }

  .signal-meta {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
  }

  .signal-label {
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .signal-dbm {
    color: var(--text-muted, #5a6d85);
    font-family: monospace;
  }

  .metrics {
    display: flex;
    gap: 0.75rem;
    justify-content: space-around;
    padding-top: 0.5rem;
    border-top: 1px solid var(--border, #2a3a54);
  }

  .metric {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .metric-val {
    font-size: 1.1rem;
    font-weight: 700;
    font-family: monospace;
    color: var(--text-primary, #e8edf5);
  }

  .metric-unit {
    font-size: 0.65rem;
    color: var(--text-muted, #5a6d85);
    text-transform: uppercase;
  }

  .no-data {
    width: 100%;
    text-align: center;
    font-size: 0.8rem;
    color: var(--text-muted, #5a6d85);
    font-style: italic;
  }
</style>
