<script lang="ts">
  import { fetchBackend } from '$lib/backend/fetch.js';
  import { notifySuccess, notifyError } from '$lib/index.js';

  let currentPassword = $state('');
  let newPassword = $state('');
  let confirmPassword = $state('');
  let changingPassword = $state(false);

  async function changePassword() {
    if (!currentPassword || !newPassword) return;
    if (newPassword !== confirmPassword) {
      notifyError('Error', 'New passwords do not match');
      return;
    }
    if (newPassword.length < 8) {
      notifyError('Error', 'Password must be at least 8 characters');
      return;
    }

    changingPassword = true;
    try {
      const resp = await fetchBackend('/auth/password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      });

      if (resp.ok) {
        notifySuccess('Success', 'Password changed successfully');
        currentPassword = '';
        newPassword = '';
        confirmPassword = '';
      } else {
        const text = await resp.text();
        notifyError('Failed', text || 'Failed to change password');
      }
    } catch {
      notifyError('Error', 'Failed to connect to server');
    } finally {
      changingPassword = false;
    }
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Settings</h1>
    <p class="page-subtitle">Manage your account and server configuration</p>
  </div>

  <div class="settings-grid">
    <div class="settings-card">
      <h2>Change Password</h2>
      <form class="form" onsubmit={(e) => { e.preventDefault(); changePassword(); }}>
        <label class="form-field">
          <span class="form-label">Current Password</span>
          <input type="password" class="form-input" bind:value={currentPassword} autocomplete="current-password" />
        </label>
        <label class="form-field">
          <span class="form-label">New Password</span>
          <input type="password" class="form-input" bind:value={newPassword} autocomplete="new-password" />
        </label>
        <label class="form-field">
          <span class="form-label">Confirm New Password</span>
          <input type="password" class="form-input" bind:value={confirmPassword} autocomplete="new-password" />
        </label>
        <button class="btn btn-primary" type="submit" disabled={changingPassword || !currentPassword || !newPassword || !confirmPassword}>
          {changingPassword ? 'Changing...' : 'Change Password'}
        </button>
      </form>
    </div>

    <div class="settings-card">
      <h2>Server Info</h2>
      <div class="info-rows">
        <div class="info-row">
          <span class="info-label">Version</span>
          <span class="info-value">Robomesh v1.0</span>
        </div>
      </div>
    </div>
  </div>
</div>

<style>
  .page {
    max-width: 800px;
  }

  .page-header {
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

  .settings-grid {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .settings-card {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 1.5rem;
  }

  .settings-card h2 {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0 0 1.25rem 0;
  }

  .form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    max-width: 400px;
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .form-label {
    font-size: 0.82rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .form-input {
    padding: 0.5rem 0.75rem;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-size: 0.9rem;
    outline: none;
    transition: border-color 0.15s;
  }

  .form-input:focus {
    border-color: var(--accent);
  }

  .btn {
    padding: 0.5rem 1.25rem;
    border: none;
    border-radius: var(--radius-sm);
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: background-color 0.12s;
    align-self: flex-start;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    background: var(--accent);
    color: #0b1120;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .info-rows {
    display: flex;
    flex-direction: column;
    gap: 0.65rem;
  }

  .info-row {
    display: flex;
    justify-content: space-between;
    font-size: 0.85rem;
  }

  .info-label {
    color: var(--text-muted);
    font-weight: 500;
  }

  .info-value {
    color: var(--text-primary);
  }
</style>
