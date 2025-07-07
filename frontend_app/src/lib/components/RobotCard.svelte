<script lang="ts">
  import type { BaseRobot } from "$lib/types.js";
  import { goto } from "$app/navigation";

  export let robot: BaseRobot;
  export let showTitle: boolean = true;
  export const btn_quickAction_label: string = "Quick";
  export const btn_quickAction: (params: any) => void = (params) => {
    // Default implementation (can be overridden)
    console.log("Quick action triggered with params:", params);
  };

  const btn_moreActions: () => void = () => {
    goto("/robots/" + robot.device_id);
  };

  // Helper function to get status color
  function getStatusColor(status: string): string {
    switch (status.toLowerCase()) {
      case "online":
      case "active":
        return "#22c55e"; // green
      case "offline":
      case "inactive":
        return "#ef4444"; // red
      case "maintenance":
      case "warning":
        return "#f59e0b"; // amber
      default:
        return "#6b7280"; // gray
    }
  }
</script>

<div class="robot-card">
  {#if showTitle}
    <h2 class="robot-name">{robot.name}</h2>
  {/if}

  <div class="robot-info">
    <ul class="robot-attributes">
      <li><strong>ID:</strong> {robot.device_id}</li>
      <li><strong>IP Address:</strong> {robot.ip}</li>
      <li><strong>Robot Type:</strong> {robot.robot_type}</li>
      <li>
        <strong>Status:</strong>
        <span
          class="status-badge"
          style="background-color: {getStatusColor(robot.status)}"
          >{robot.status}</span
        >
      </li>
      <li><strong>Last Seen:</strong> {robot.last_seen ? new Date(robot.last_seen * 1000).toLocaleString() : 'Unknown'}</li>
    </ul>
  </div>

  <!-- Optional slot for additional content -->
  <div class="robot-actions">
    <button on:click={() => btn_quickAction(robot)}>{btn_quickAction_label}</button>
    <button on:click={btn_moreActions}>More</button>
  </div>
</div>

<style>
  .robot-card {
    border: 1px solid #e5e7eb;
    border-radius: 12px;
    padding: 20px;
    margin: 16px 0;
    background: white;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    transition: box-shadow 0.2s ease;
    width: fit-content;
    min-width: 280px;
    max-width: 400px;
  }

  .robot-card:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  }

  .robot-name {
    font-size: 1.5rem;
    font-weight: 600;
    margin-bottom: 16px;
    color: #1f2937;
    border-bottom: 2px solid #f3f4f6;
    padding-bottom: 8px;
  }

  .robot-info {
    margin-bottom: 16px;
  }

  .robot-attributes {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .robot-attributes li {
    padding: 8px 0;
    display: flex;
    align-items: center;
    border-bottom: 1px solid #f9fafb;
  }

  .robot-attributes li:last-child {
    border-bottom: none;
  }

  .robot-attributes strong {
    color: #374151;
    min-width: 100px;
    margin-right: 8px;
  }

  .status-badge {
    display: inline-block;
    color: white;
    padding: 4px 8px;
    border-radius: 12px;
    font-size: 0.875rem;
    font-weight: 500;
    text-transform: capitalize;
  }

  .robot-actions {
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid #f3f4f6;
    display: flex;
    gap: 8px;
    justify-content: center;
  }

  .robot-actions button {
    flex: 1;
    padding: 8px 16px;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    background-color: #f9fafb;
    color: #374151;
    cursor: pointer;
    font-weight: 500;
    transition: background-color 0.2s ease;
  }

  .robot-actions button:hover {
    background-color: #e5e7eb;
  }

  .robot-actions button:first-child {
    background-color: #3b82f6;
    color: white;
    border-color: #2563eb;
  }

  .robot-actions button:first-child:hover {
    background-color: #2563eb;
  }
</style>
