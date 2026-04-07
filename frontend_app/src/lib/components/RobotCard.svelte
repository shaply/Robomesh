<script lang="ts">
  import type { BaseRobot } from "$lib/types.js";
  import { goto } from "$app/navigation";
  import { getHandlerStatus, startHandler, killHandler } from "$lib/backend/get_robots.js";
  import { onMount } from "svelte";

  let {
    robot,
    showTitle = true,
  }: {
    robot: BaseRobot;
    showTitle?: boolean;
  } = $props();

  let handlerActive = $state(false);
  let handlerLoading = $state(false);

  const btn_moreActions: () => void = () => {
    goto("/robots/" + robot.device_id);
  };

  onMount(async () => {
    try {
      const status = await getHandlerStatus(robot.device_id);
      if (status) {
        handlerActive = status.active;
      }
    } catch (e) {
      console.error('Failed to fetch handler status:', e);
    }
  });

  async function toggleHandler() {
    if (handlerLoading) return;
    handlerLoading = true;
    try {
      if (handlerActive) {
        const ok = await killHandler(robot.device_id);
        if (ok) handlerActive = false;
      } else {
        const ok = await startHandler(robot.device_id);
        if (ok) handlerActive = true;
      }
    } finally {
      handlerLoading = false;
    }
  }

  function getStatusColor(status: string): string {
    switch (status.toLowerCase()) {
      case "online":
      case "active":
        return "var(--success)";
      case "offline":
      case "inactive":
        return "var(--error)";
      case "maintenance":
      case "warning":
        return "var(--warning)";
      default:
        return "var(--text-muted)";
    }
  }

  function getStatusBg(status: string): string {
    switch (status.toLowerCase()) {
      case "online":
      case "active":
        return "var(--success-muted)";
      case "offline":
      case "inactive":
        return "var(--error-muted)";
      case "maintenance":
      case "warning":
        return "var(--warning-muted)";
      default:
        return "var(--bg-hover)";
    }
  }
</script>

<div class="card">
  <div class="card-top">
    {#if showTitle}
      <h3 class="card-name">{robot.name}</h3>
    {/if}
    <div class="status-pills">
      <span
        class="status-pill"
        style="color: {getStatusColor(robot.status)}; background: {getStatusBg(robot.status)};"
      >
        <span class="status-dot" style="background: {getStatusColor(robot.status)};"></span>
        {robot.status}
      </span>
      <span
        class="status-pill"
        style="color: {handlerActive ? 'var(--success)' : 'var(--text-muted)'}; background: {handlerActive ? 'var(--success-muted)' : 'var(--bg-hover)'};"
      >
        <span class="status-dot" style="background: {handlerActive ? 'var(--success)' : 'var(--text-muted)'};"></span>
        Handler {handlerActive ? 'on' : 'off'}
      </span>
    </div>
  </div>

  <div class="card-details">
    <div class="detail-row">
      <span class="detail-label">ID</span>
      <span class="detail-value mono">{robot.device_id}</span>
    </div>
    <div class="detail-row">
      <span class="detail-label">IP</span>
      <span class="detail-value mono">{robot.ip}</span>
    </div>
    <div class="detail-row">
      <span class="detail-label">Type</span>
      <span class="detail-value">{robot.robot_type}</span>
    </div>
    <div class="detail-row">
      <span class="detail-label">Last seen</span>
      <span class="detail-value">{robot.last_seen ? new Date(robot.last_seen * 1000).toLocaleString() : 'Unknown'}</span>
    </div>
  </div>

  <div class="card-actions">
    <button
      class="btn-handler"
      class:btn-handler-active={handlerActive}
      onclick={toggleHandler}
      disabled={handlerLoading}
    >
      {handlerLoading ? '...' : handlerActive ? 'Kill Handler' : 'Start Handler'}
    </button>
    <button class="btn-more" onclick={btn_moreActions}>Details</button>
  </div>
</div>

<style>
  .card {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.25rem;
    transition: border-color 0.15s, box-shadow 0.15s;
  }

  .card:hover {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent-muted);
  }

  .card-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1rem;
    gap: 0.5rem;
  }

  .card-name {
    font-size: 1.05rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
    letter-spacing: -0.01em;
  }

  .status-pills {
    display: flex;
    gap: 0.35rem;
    flex-shrink: 0;
  }

  .status-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 3px 10px;
    border-radius: 20px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: capitalize;
    white-space: nowrap;
  }

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .card-details {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-bottom: 1.15rem;
  }

  .detail-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 0.85rem;
  }

  .detail-label {
    color: var(--text-muted);
    font-weight: 500;
  }

  .detail-value {
    color: var(--text-secondary);
  }

  .detail-value.mono {
    font-family: var(--font-mono);
    font-size: 0.8rem;
  }

  .card-actions {
    display: flex;
    gap: 0.5rem;
    padding-top: 1rem;
    border-top: 1px solid var(--border);
  }

  .card-actions button {
    flex: 1;
    padding: 0.5rem 0.75rem;
    border-radius: var(--radius-sm);
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
    transition: background-color 0.12s;
    border: none;
  }

  .card-actions button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-handler {
    background: var(--accent);
    color: #0b1120;
  }

  .btn-handler:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .btn-handler-active {
    background: var(--error-muted, #3b1a1a);
    color: var(--error, #ef4444);
  }

  .btn-handler-active:hover:not(:disabled) {
    background: var(--error, #ef4444);
    color: white;
  }

  .btn-more {
    background: var(--bg-hover);
    color: var(--text-secondary);
    border: 1px solid var(--border) !important;
  }

  .btn-more:hover {
    background: var(--border);
    color: var(--text-primary);
  }

  @media (max-width: 480px) {
    .card-top {
      flex-direction: column;
      align-items: flex-start;
    }

    .status-pills {
      width: 100%;
      justify-content: flex-start;
    }

    .detail-value.mono {
      font-size: 0.72rem;
      word-break: break-all;
    }

    .card-actions {
      flex-direction: column;
    }
  }
</style>
