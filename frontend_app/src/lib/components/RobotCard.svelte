<script lang="ts">
  import type { BaseRobot } from "$lib/types.js";
  import { goto } from "$app/navigation";

  export let robot: BaseRobot;
  export let showTitle: boolean = true;
  export const btn_quickAction_label: string = "Quick";
  export const btn_quickAction: (params: any) => void = (params) => {
    console.log("Quick action triggered with params:", params);
  };

  const btn_moreActions: () => void = () => {
    goto("/robots/" + robot.device_id);
  };

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
    <span
      class="status-pill"
      style="color: {getStatusColor(robot.status)}; background: {getStatusBg(robot.status)};"
    >
      <span class="status-dot" style="background: {getStatusColor(robot.status)};"></span>
      {robot.status}
    </span>
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
    <button class="btn-quick" on:click={() => btn_quickAction(robot)}>{btn_quickAction_label}</button>
    <button class="btn-more" on:click={btn_moreActions}>Details</button>
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
  }

  .card-name {
    font-size: 1.05rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
    letter-spacing: -0.01em;
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

  .btn-quick {
    background: var(--accent);
    color: #0b1120;
  }

  .btn-quick:hover {
    background: var(--accent-hover);
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
</style>
