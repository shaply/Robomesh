<script lang="ts">
  import { getRobotComponentAsync, getRobotComponent } from '$lib/robots/registry.js';
  import { fetchBackend } from '$lib/backend/fetch.js';
  import { getHandlerStatus, startHandler, killHandler } from '$lib/backend/get_robots.js';
  import { onMount, onDestroy } from 'svelte';

  let { data } = $props();

  // Start with sync fallback, then try async plugin load
  let robotConfig = $state(getRobotComponent('default'));
  let RobotComponent = $derived(robotConfig.component);
  let HandlerComponent = $derived(robotConfig.handlerComponent);
  let loading = $state(true);
  let activeView = $state<'handler' | 'info' | 'logs'>('info');

  // Robot detail state
  let detail = $state<any>(null);
  let handlerActive = $state(false);
  let handlerLoading = $state(false);
  let logLines = $state<string[]>([]);
  let logEventSource: EventSource | null = null;
  let detailInterval: ReturnType<typeof setInterval>;

  onMount(async () => {
    const robotType = data.robot?.robot_type;
    if (robotType) {
      robotConfig = getRobotComponent(robotType);
      robotConfig = await getRobotComponentAsync(robotType);
    }
    loading = false;

    await fetchDetail();
    detailInterval = setInterval(fetchDetail, 15000);
  });

  onDestroy(() => {
    if (detailInterval) clearInterval(detailInterval);
    stopLogStream();
  });

  async function fetchDetail() {
    if (!data.robot) return;
    try {
      const resp = await fetchBackend(`/robot/${data.robot.device_id}`);
      if (resp.ok) {
        detail = await resp.json();
        handlerActive = detail?.handler?.active ?? false;
      }
    } catch (e) {
      console.error('Failed to fetch robot detail:', e);
    }
  }

  async function toggleHandler() {
    if (handlerLoading || !data.robot) return;
    handlerLoading = true;
    try {
      if (handlerActive) {
        const ok = await killHandler(data.robot.device_id);
        if (ok) {
          handlerActive = false;
          stopLogStream();
        }
      } else {
        const ok = await startHandler(data.robot.device_id);
        if (ok) handlerActive = true;
      }
      await fetchDetail();
    } finally {
      handlerLoading = false;
    }
  }

  async function startLogStream() {
    if (!data.robot || logEventSource) return;
    logLines = [];

    // Fetch a single-use ticket for SSE auth (EventSource can't send headers)
    let ticket = '';
    try {
      const ticketRes = await fetchBackend('/auth/ticket', { method: 'POST' });
      if (ticketRes.ok) {
        const ticketData = await ticketRes.json();
        ticket = ticketData.ticket;
      } else {
        logLines = ['[system] Failed to authenticate log stream'];
        return;
      }
    } catch {
      logLines = ['[system] Failed to authenticate log stream'];
      return;
    }

    const url = `http://${location.hostname}:${import.meta.env.VITE_BACKEND_PORT || '8080'}/handler/${data.robot.device_id}/logs?ticket=${encodeURIComponent(ticket)}`;
    logEventSource = new EventSource(url);
    logEventSource.onmessage = (e) => {
      try {
        const log = JSON.parse(e.data);
        logLines = [...logLines.slice(-499), `[${log.stream}] ${log.line}`];
      } catch {
        logLines = [...logLines.slice(-499), e.data];
      }
    };
    logEventSource.onerror = () => {
      stopLogStream();
    };
  }

  function stopLogStream() {
    if (logEventSource) {
      logEventSource.close();
      logEventSource = null;
    }
  }

  function formatTimestamp(unix: number): string {
    if (!unix) return 'Unknown';
    return new Date(unix * 1000).toLocaleString();
  }

  function timeSince(unix: number): string {
    if (!unix) return 'Unknown';
    const seconds = Math.floor(Date.now() / 1000 - unix);
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
  }

  $effect(() => {
    if (activeView === 'logs' && handlerActive) {
      startLogStream();
    } else {
      stopLogStream();
    }
  });
</script>

<div class="robot-page">
  {#if data.robot}
    <div class="robot-header">
      <div class="header-left">
        <h1>{robotConfig.displayName}</h1>
        {#if robotConfig.description}
          <p class="description">{robotConfig.description}</p>
        {/if}
        <div class="header-badges">
          {#if robotConfig.isPlugin}
            <span class="badge badge-plugin">Plugin</span>
          {/if}
          {#if detail?.online}
            <span class="badge badge-online">Online</span>
          {:else}
            <span class="badge badge-offline">Offline</span>
          {/if}
          {#if handlerActive}
            <span class="badge badge-handler">Handler Active</span>
          {/if}
        </div>
      </div>
      <div class="header-actions">
        <button
          class="btn"
          class:btn-danger={handlerActive}
          class:btn-primary={!handlerActive}
          onclick={toggleHandler}
          disabled={handlerLoading}
        >
          {handlerLoading ? '...' : handlerActive ? 'Kill Handler' : 'Start Handler'}
        </button>
      </div>
    </div>

    <div class="view-tabs">
      <button class="tab" class:active={activeView === 'info'} onclick={() => activeView = 'info'}>
        Info
      </button>
      {#if HandlerComponent && !loading}
        <button class="tab" class:active={activeView === 'handler'} onclick={() => activeView = 'handler'}>
          Handler
        </button>
      {/if}
      <button
        class="tab" class:active={activeView === 'logs'}
        onclick={() => activeView = 'logs'}
        disabled={!handlerActive}
      >
        Logs
      </button>
    </div>

    {#if loading}
      <div class="loading">
        <span class="spinner"></span>
        Loading...
      </div>
    {:else if activeView === 'info'}
      <div class="info-grid">
        <!-- Connection Info -->
        <div class="info-card">
          <h3>Connection</h3>
          <div class="info-rows">
            <div class="info-row">
              <span class="info-label">UUID</span>
              <span class="info-value mono">{data.robot.device_id}</span>
            </div>
            <div class="info-row">
              <span class="info-label">IP</span>
              <span class="info-value mono">{detail?.ip || data.robot.ip || 'Unknown'}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Type</span>
              <span class="info-value">{data.robot.robot_type}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Connected</span>
              <span class="info-value">{detail?.connected_at ? formatTimestamp(detail.connected_at) : 'N/A'}</span>
            </div>
          </div>
        </div>

        <!-- Heartbeat Info -->
        <div class="info-card">
          <h3>Heartbeat</h3>
          {#if detail?.heartbeat}
            <div class="info-rows">
              <div class="info-row">
                <span class="info-label">Last Seen</span>
                <span class="info-value">{timeSince(detail.heartbeat.last_seen)}</span>
              </div>
              <div class="info-row">
                <span class="info-label">Sequence</span>
                <span class="info-value mono">{detail.heartbeat.last_seq}</span>
              </div>
              <div class="info-row">
                <span class="info-label">Heartbeat IP</span>
                <span class="info-value mono">{detail.heartbeat.ip}</span>
              </div>
            </div>
          {:else}
            <div class="info-empty">No heartbeat data</div>
          {/if}
        </div>

        <!-- Handler Info -->
        <div class="info-card">
          <h3>Handler</h3>
          {#if detail?.handler?.active}
            <div class="info-rows">
              <div class="info-row">
                <span class="info-label">Status</span>
                <span class="info-value badge-inline badge-online">Running</span>
              </div>
              <div class="info-row">
                <span class="info-label">PID</span>
                <span class="info-value mono">{detail.handler.pid}</span>
              </div>
              <div class="info-row">
                <span class="info-label">Type</span>
                <span class="info-value">{detail.handler.device_type}</span>
              </div>
            </div>
          {:else}
            <div class="info-empty">No handler running</div>
          {/if}
        </div>

        <!-- Registration Info -->
        <div class="info-card">
          <h3>Registration</h3>
          {#if detail?.registered}
            <div class="info-rows">
              <div class="info-row">
                <span class="info-label">Status</span>
                <span class="info-value badge-inline" class:badge-online={!detail.registration?.is_blacklisted} class:badge-danger={detail.registration?.is_blacklisted}>
                  {detail.registration?.is_blacklisted ? 'Blacklisted' : 'Registered'}
                </span>
              </div>
              <div class="info-row">
                <span class="info-label">Registered</span>
                <span class="info-value">{detail.registration?.created_at ? new Date(detail.registration.created_at).toLocaleString() : 'Unknown'}</span>
              </div>
            </div>
          {:else}
            <div class="info-empty">Not permanently registered (ephemeral)</div>
          {/if}
        </div>
      </div>
    {:else if activeView === 'handler' && HandlerComponent}
      <HandlerComponent robot={data.robot} {fetchBackend} sendToHandler={async (msg: string) => {
        if (!data.robot) return;
        await fetchBackend(`/robot/${data.robot.device_id}/message`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ message: msg })
        });
      }} />
    {:else if activeView === 'logs'}
      <div class="logs-panel">
        {#if !handlerActive}
          <div class="info-empty">Start a handler to view logs</div>
        {:else if logLines.length === 0}
          <div class="info-empty">Waiting for log output...</div>
        {:else}
          <pre class="log-output">{logLines.join('\n')}</pre>
        {/if}
      </div>
    {/if}
  {:else}
    <div class="error">
      <h2>Robot not found</h2>
      <p>The robot you're looking for doesn't exist or couldn't be loaded.</p>
    </div>
  {/if}
</div>

<style>
  .robot-page {
    max-width: 1200px;
    margin: 0 auto;
  }

  .robot-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1.5rem;
    gap: 1rem;
  }

  .header-left h1 {
    color: var(--text-primary);
    margin: 0 0 0.25rem 0;
    font-size: 1.5rem;
  }

  .description {
    color: var(--text-secondary);
    font-size: 0.9rem;
    margin: 0 0 0.5rem 0;
  }

  .header-badges {
    display: flex;
    gap: 0.35rem;
    flex-wrap: wrap;
  }

  .badge {
    display: inline-flex;
    align-items: center;
    padding: 2px 10px;
    border-radius: 20px;
    font-size: 0.72rem;
    font-weight: 600;
    text-transform: uppercase;
  }

  .badge-plugin { background: var(--accent-muted); color: var(--accent); }
  .badge-online { background: var(--success-muted); color: var(--success); }
  .badge-offline { background: var(--error-muted); color: var(--error); }
  .badge-handler { background: var(--accent-muted); color: var(--accent); }
  .badge-danger { background: var(--error-muted); color: var(--error); }

  .badge-inline {
    display: inline;
    padding: 2px 8px;
    border-radius: 12px;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .header-actions {
    flex-shrink: 0;
  }

  .btn {
    padding: 0.5rem 1.25rem;
    border: none;
    border-radius: var(--radius-sm);
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: background-color 0.12s;
  }

  .btn:disabled { opacity: 0.5; cursor: not-allowed; }
  .btn-primary { background: var(--accent); color: #0b1120; }
  .btn-primary:hover:not(:disabled) { background: var(--accent-hover); }
  .btn-danger { background: var(--error-muted); color: var(--error); }
  .btn-danger:hover:not(:disabled) { background: var(--error); color: white; }

  .view-tabs {
    display: flex;
    gap: 0.25rem;
    margin-bottom: 1.25rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0;
  }

  .tab {
    padding: 0.5rem 1rem;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: 500;
    border-bottom: 2px solid transparent;
    transition: color 0.12s;
  }

  .tab:hover { color: var(--text-primary); }
  .tab.active { color: var(--text-primary); border-bottom-color: var(--accent); }
  .tab:disabled { opacity: 0.4; cursor: not-allowed; }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1rem;
  }

  .info-card {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.25rem;
  }

  .info-card h3 {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin: 0 0 1rem 0;
  }

  .info-rows {
    display: flex;
    flex-direction: column;
    gap: 0.65rem;
  }

  .info-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 0.85rem;
  }

  .info-label { color: var(--text-muted); font-weight: 500; }
  .info-value { color: var(--text-primary); }
  .info-value.mono { font-family: var(--font-mono); font-size: 0.8rem; }
  .info-empty {
    color: var(--text-muted);
    font-size: 0.85rem;
    text-align: center;
    padding: 1rem 0;
  }

  .logs-panel {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    min-height: 300px;
    max-height: 600px;
    overflow: auto;
  }

  .log-output {
    margin: 0;
    padding: 1rem;
    font-family: var(--font-mono);
    font-size: 0.78rem;
    line-height: 1.6;
    color: var(--text-secondary);
    white-space: pre-wrap;
    word-break: break-all;
  }

  .loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    padding: 3rem;
    color: var(--text-secondary);
  }

  .spinner {
    width: 20px;
    height: 20px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin { to { transform: rotate(360deg); } }

  .error {
    text-align: center;
    padding: 2rem;
    background: var(--bg-surface);
    border: 1px solid var(--error);
    border-radius: 0.5rem;
    color: var(--error);
  }

  /* Mobile */
  @media (max-width: 768px) {
    .robot-header { flex-direction: column; }
    .info-grid { grid-template-columns: 1fr; }
  }
</style>
