<script lang="ts">
  import { fetchBackend } from "$lib/backend/fetch.js";
  import { removeNotification } from "$lib/index.js";
  import type { RegisteringRobotEvent } from "$lib/types.js";

  // Props are automatically passed by the notification system
  let {
    notificationId,
    registering_robot,
  }: {
    notificationId: string;
    registering_robot: RegisteringRobotEvent;
  } = $props();

  // State to track the registration process
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

    // Update state to show we're processing
    stateNotif = "processing";

    try {
      const response = await fetchBackend("/robot/register", {
        method: "POST",
        body: JSON.stringify({
          registering_robot: registering_robot,
          accept: accept ? "yes" : "no",
        }),
      });

      if (!response.ok) {
        stateNotif = "error";
        errorMessage = `Failed to ${accept ? "accept" : "reject"} robot: ${response.statusText}`;
        return;
      }

      // Success - update state and close notification after a brief delay
      stateNotif = "success";
      setTimeout(() => {
        removeNotification(notificationId);
      }, 2000); // Show success message for 2 seconds before closing

    } catch (error) {
      stateNotif = "error";
      errorMessage = `Network error: ${error instanceof Error ? error.message : "Unknown error"}`;
    }
  }
</script>

<div class="notification-registering-robot">
  {#if stateNotif === "pending"}
    <div class="robot-info">
      <p><strong>Type:</strong> {registering_robot.robot_type}</p>
      <p><strong>Device ID:</strong> {registering_robot.device_id}</p>
      <p><strong>IP Address:</strong> {registering_robot.ip}</p>
    </div>
    <div class="actions">
      <button onclick={() => registerRobot(registering_robot, true)} class="accept-btn">
        Accept
      </button>
      <button onclick={() => registerRobot(registering_robot, false)} class="reject-btn">
        Reject
      </button>
    </div>
  {:else if stateNotif === "processing"}
    <div class="processing">
      <p>Processing registration...</p>
      <div class="spinner"></div>
    </div>
  {:else if stateNotif === "success"}
    <div class="success">
      <p>✅ Robot registration completed successfully!</p>
    </div>
  {:else if stateNotif === "error"}
    <div class="error">
      <p>❌ {errorMessage}</p>
      <button onclick={() => stateNotif = "pending"} class="retry-btn">
        Try Again
      </button>
    </div>
  {/if}
</div>

<style>
  .notification-registering-robot {
    padding: 1rem;
  }

  .robot-info {
    margin-bottom: 1rem;
  }

  .robot-info p {
    margin: 0.25rem 0;
    font-size: 0.9rem;
  }

  .actions {
    display: flex;
    gap: 0.5rem;
  }

  .accept-btn {
    background-color: #22c55e;
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 0.25rem;
    cursor: pointer;
    font-weight: 500;
  }

  .accept-btn:hover {
    background-color: #16a34a;
  }

  .reject-btn {
    background-color: #ef4444;
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 0.25rem;
    cursor: pointer;
    font-weight: 500;
  }

  .reject-btn:hover {
    background-color: #dc2626;
  }

  .retry-btn {
    background-color: #3b82f6;
    color: white;
    border: none;
    padding: 0.25rem 0.75rem;
    border-radius: 0.25rem;
    cursor: pointer;
    font-size: 0.875rem;
    margin-top: 0.5rem;
  }

  .retry-btn:hover {
    background-color: #2563eb;
  }

  .processing {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .spinner {
    width: 1rem;
    height: 1rem;
    border: 2px solid #e5e7eb;
    border-top: 2px solid #3b82f6;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }

  .success {
    color: #16a34a;
  }

  .error {
    color: #dc2626;
  }

  .error p {
    margin: 0 0 0.5rem 0;
  }
</style>
