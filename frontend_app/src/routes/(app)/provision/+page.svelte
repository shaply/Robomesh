<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { getRegisteredRobots, provisionRobot, blacklistRobot } from "$lib/backend/get_robots.js";
  import type { RegisteredRobot } from "$lib/types.js";
  import { notifySuccess, notifyError } from "$lib/index.js";
  import PageButton from "$lib/components/page-button.svelte";

  let robots: RegisteredRobot[] = [];
  let loading = true;
  let error: string | null = null;
  let intervalId: number;

  let showForm = false;
  let formUuid = "";
  let formPublicKey = "";
  let formDeviceType = "";
  let formSubmitting = false;

  async function fetchRobots() {
    try {
      loading = true;
      error = null;
      const result = await getRegisteredRobots();
      if (!result) throw new Error("Failed to fetch registered robots");
      robots = result;
    } catch (err) {
      console.error("Error fetching robots:", err);
      error = err instanceof Error ? err.message : "Failed to fetch robots";
      robots = [];
    } finally {
      loading = false;
    }
  }

  async function handleProvision() {
    if (!formUuid || !formPublicKey || !formDeviceType) {
      notifyError("Validation Error", "All fields are required");
      return;
    }

    formSubmitting = true;
    const success = await provisionRobot(formUuid, formPublicKey, formDeviceType);
    formSubmitting = false;

    if (success) {
      notifySuccess("Robot Provisioned", `${formUuid} registered successfully`);
      formUuid = "";
      formPublicKey = "";
      formDeviceType = "";
      showForm = false;
      fetchRobots();
    } else {
      notifyError("Provision Failed", "Could not register robot. Check if UUID already exists.");
    }
  }

  async function handleBlacklist(uuid: string, current: boolean) {
    const success = await blacklistRobot(uuid, !current);
    if (success) {
      notifySuccess("Updated", `${uuid} ${!current ? "blacklisted" : "unblacklisted"}`);
      fetchRobots();
    } else {
      notifyError("Failed", "Could not update blacklist status");
    }
  }

  onMount(() => {
    fetchRobots();
    intervalId = setInterval(fetchRobots, 30000);
  });

  onDestroy(() => {
    if (intervalId) clearInterval(intervalId);
  });
</script>

<div class="page">
  <div class="page-header">
    <div>
      <h1>Robot Registry</h1>
      <p class="page-subtitle">Permanent robot provisioning and key management</p>
    </div>
    <div class="header-actions">
      <PageButton on:click={() => (showForm = !showForm)}>
        {showForm ? "Cancel" : "+ Add Robot"}
      </PageButton>
      <PageButton on:click={fetchRobots} variant="secondary">Refresh</PageButton>
    </div>
  </div>

  {#if showForm}
    <div class="provision-form">
      <div class="form-header">
        <h2>Provision New Robot</h2>
        <p>Register a robot's public key so it can authenticate via TCP using the AUTH command.</p>
      </div>
      <div class="form-grid">
        <div class="form-field">
          <label for="uuid">Robot UUID</label>
          <input id="uuid" type="text" bind:value={formUuid} placeholder="e.g., robot-001" />
        </div>
        <div class="form-field">
          <label for="device-type">Device Type</label>
          <input id="device-type" type="text" bind:value={formDeviceType} placeholder="e.g., example_robot" />
        </div>
        <div class="form-field full-width">
          <label for="public-key">Public Key (hex)</label>
          <textarea id="public-key" bind:value={formPublicKey} placeholder="Ed25519 public key in hex format" rows="3"></textarea>
        </div>
      </div>
      <div class="form-actions">
        <button class="btn-register" on:click={handleProvision} disabled={formSubmitting}>
          {formSubmitting ? "Registering..." : "Register Robot"}
        </button>
      </div>
    </div>
  {/if}

  <div class="table-wrapper">
    {#if loading}
      <div class="state-msg">
        <span class="spinner"></span>
        Loading registered robots...
      </div>
    {:else if error}
      <div class="state-msg state-error">Error: {error}</div>
    {:else if robots.length === 0}
      <div class="state-msg">
        <span class="empty-label">No registered robots</span>
        <span class="empty-hint">Use "+ Add Robot" to provision one</span>
      </div>
    {:else}
      <table>
        <thead>
          <tr>
            <th>UUID</th>
            <th>Device Type</th>
            <th>Public Key</th>
            <th>Status</th>
            <th>Created</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each robots as robot (robot.UUID)}
            <tr class:row-blacklisted={robot.IsBlacklisted}>
              <td class="cell-mono">{robot.UUID}</td>
              <td>{robot.DeviceType}</td>
              <td class="cell-key" title={robot.PublicKey}>
                {robot.PublicKey.slice(0, 16)}...
              </td>
              <td>
                {#if robot.IsBlacklisted}
                  <span class="badge badge-danger">Blacklisted</span>
                {:else}
                  <span class="badge badge-ok">Active</span>
                {/if}
              </td>
              <td class="cell-date">{new Date(robot.CreatedAt).toLocaleString()}</td>
              <td>
                <button
                  class="action-btn"
                  class:btn-blacklist={!robot.IsBlacklisted}
                  class:btn-unblacklist={robot.IsBlacklisted}
                  on:click={() => handleBlacklist(robot.UUID, robot.IsBlacklisted)}
                >
                  {robot.IsBlacklisted ? "Unblacklist" : "Blacklist"}
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>

<style>
  .page {
    max-width: 1200px;
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1.75rem;
  }

  .page-header h1 {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
    margin: 0 0 0.25rem 0;
    letter-spacing: -0.02em;
  }

  .page-subtitle {
    color: var(--text-secondary);
    font-size: 0.9rem;
    margin: 0;
  }

  .header-actions {
    display: flex;
    gap: 0.35rem;
    flex-shrink: 0;
  }

  /* --- Provision form --- */
  .provision-form {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 1.5rem;
    margin-bottom: 1.5rem;
  }

  .form-header {
    margin-bottom: 1.25rem;
  }

  .form-header h2 {
    font-size: 1.1rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0 0 0.25rem 0;
  }

  .form-header p {
    color: var(--text-secondary);
    font-size: 0.85rem;
    margin: 0;
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }

  .full-width {
    grid-column: 1 / -1;
  }

  .form-field label {
    display: block;
    font-weight: 500;
    font-size: 0.85rem;
    color: var(--text-secondary);
    margin-bottom: 0.35rem;
  }

  .form-field input,
  .form-field textarea {
    width: 100%;
    padding: 0.55rem 0.75rem;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    font-size: 0.9rem;
    font-family: var(--font-mono);
    color: var(--text-primary);
    outline: none;
    transition: border-color 0.15s;
  }

  .form-field input::placeholder,
  .form-field textarea::placeholder {
    color: var(--text-muted);
  }

  .form-field input:focus,
  .form-field textarea:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-muted);
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 1.25rem;
  }

  .btn-register {
    background: var(--success);
    color: white;
    border: none;
    padding: 0.55rem 1.5rem;
    border-radius: var(--radius-sm);
    cursor: pointer;
    font-weight: 600;
    font-size: 0.9rem;
    transition: background-color 0.12s;
  }

  .btn-register:hover:not(:disabled) {
    background: #16a34a;
  }

  .btn-register:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* --- Table --- */
  .table-wrapper {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    overflow: hidden;
  }

  .state-msg {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    padding: 3rem 1rem;
    color: var(--text-secondary);
    font-size: 0.95rem;
  }

  .state-error {
    color: var(--error);
  }

  .empty-label {
    font-weight: 600;
  }

  .empty-hint {
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .spinner {
    width: 20px;
    height: 20px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  thead {
    background: var(--bg-elevated);
  }

  th {
    text-align: left;
    padding: 0.7rem 1rem;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    border-bottom: 1px solid var(--border);
  }

  td {
    padding: 0.7rem 1rem;
    font-size: 0.875rem;
    border-bottom: 1px solid var(--border-light);
    color: var(--text-secondary);
  }

  tr:last-child td {
    border-bottom: none;
  }

  tr:hover {
    background: var(--bg-elevated);
  }

  .row-blacklisted {
    background: var(--error-muted);
  }

  .row-blacklisted:hover {
    background: rgba(239, 68, 68, 0.15);
  }

  .cell-mono {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .cell-key {
    font-family: var(--font-mono);
    font-size: 0.75rem;
    color: var(--text-muted);
    cursor: help;
  }

  .cell-date {
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .badge {
    display: inline-block;
    padding: 2px 10px;
    border-radius: 20px;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .badge-ok {
    background: var(--success-muted);
    color: var(--success);
  }

  .badge-danger {
    background: var(--error-muted);
    color: var(--error);
  }

  .action-btn {
    padding: 4px 12px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    cursor: pointer;
    font-size: 0.78rem;
    font-weight: 500;
    background: var(--bg-elevated);
    color: var(--text-secondary);
    transition: background-color 0.12s, color 0.12s;
  }

  .btn-blacklist {
    border-color: rgba(239, 68, 68, 0.3);
    color: var(--error);
  }

  .btn-blacklist:hover {
    background: var(--error-muted);
  }

  .btn-unblacklist {
    border-color: rgba(34, 197, 94, 0.3);
    color: var(--success);
  }

  .btn-unblacklist:hover {
    background: var(--success-muted);
  }
</style>
