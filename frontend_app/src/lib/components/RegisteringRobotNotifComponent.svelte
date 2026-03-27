<script lang="ts">
  import { fetchBackend } from "$lib/backend/fetch.js";
  import { removeNotification } from "$lib/index.js";
  import type { RegisteringRobotEvent } from "$lib/types.js";

  let {
    notificationId,
    registering_robot,
  }: {
    notificationId: string;
    registering_robot: RegisteringRobotEvent;
  } = $props();

  let stateNotif = $state<"pending" | "processing" | "success" | "error">("pending");
  let errorMessage = $state<string | null>(null);

  async function registerRobot(
    registering_robot: RegisteringRobotEvent,
    accept: boolean
  ) {
    if (!registering_robot) {
      stateNotif = "error";
      errorMessage = "Invalid robot data";
      return;
    }

    stateNotif = "processing";

    try {
      const response = await fetchBackend("/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          uuid: registering_robot.device_id,
          accept: accept,
        }),
      });

      if (!response.ok) {
        stateNotif = "error";
        errorMessage = `Failed to ${accept ? "accept" : "reject"} robot: ${response.statusText}`;
        return;
      }

      stateNotif = "success";
      setTimeout(() => {
        removeNotification(notificationId);
      }, 2000);

    } catch (error) {
      stateNotif = "error";
      errorMessage = `Network error: ${error instanceof Error ? error.message : "Unknown error"}`;
    }
  }
</script>

<div class="reg-notif">
  {#if stateNotif === "pending"}
    <div class="robot-info">
      <div class="info-row"><span class="info-label">Type</span> <span>{registering_robot.robot_type}</span></div>
      <div class="info-row"><span class="info-label">ID</span> <span class="mono">{registering_robot.device_id}</span></div>
      <div class="info-row"><span class="info-label">IP</span> <span class="mono">{registering_robot.ip}</span></div>
    </div>
    <div class="actions">
      <button onclick={() => registerRobot(registering_robot, true)} class="btn-accept">Accept</button>
      <button onclick={() => registerRobot(registering_robot, false)} class="btn-reject">Reject</button>
    </div>
  {:else if stateNotif === "processing"}
    <div class="processing">
      <div class="spinner"></div>
      <span>Processing...</span>
    </div>
  {:else if stateNotif === "success"}
    <div class="result-success">Registration completed</div>
  {:else if stateNotif === "error"}
    <div class="result-error">
      <span>{errorMessage}</span>
      <button onclick={() => stateNotif = "pending"} class="btn-retry">Retry</button>
    </div>
  {/if}
</div>

<style>
  .reg-notif {
    padding: 0.5rem 0 0 0;
  }

  .robot-info {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    margin-bottom: 0.75rem;
  }

  .info-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.8rem;
  }

  .info-label {
    color: var(--text-muted);
    font-weight: 500;
    min-width: 32px;
  }

  .mono {
    font-family: var(--font-mono);
    font-size: 0.78rem;
  }

  .actions {
    display: flex;
    gap: 0.4rem;
  }

  .btn-accept,
  .btn-reject,
  .btn-retry {
    border: none;
    padding: 0.35rem 0.85rem;
    border-radius: var(--radius-sm);
    cursor: pointer;
    font-weight: 500;
    font-size: 0.8rem;
    transition: background-color 0.12s;
  }

  .btn-accept {
    background: var(--success);
    color: white;
  }
  .btn-accept:hover {
    background: #16a34a;
  }

  .btn-reject {
    background: var(--error);
    color: white;
  }
  .btn-reject:hover {
    background: #dc2626;
  }

  .btn-retry {
    background: var(--accent);
    color: #0b1120;
    margin-top: 0.4rem;
  }
  .btn-retry:hover {
    background: var(--accent-hover);
  }

  .processing {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--text-secondary);
    font-size: 0.8rem;
  }

  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .result-success {
    color: var(--success);
    font-size: 0.8rem;
    font-weight: 500;
  }

  .result-error {
    color: var(--error);
    font-size: 0.8rem;
  }
</style>
