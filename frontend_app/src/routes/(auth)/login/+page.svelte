<script lang="ts">
  import { goto } from "$app/navigation";
  import { fetchBackend } from "$lib/backend/fetch.js";
  import { API_AUTH_TOKEN } from "$lib/const.js";

  let username = "";
  let password = "";
  let loading = false;
  let error = "";

  async function handleLogin() {
    if (!username || !password) {
      error = "Please enter both username and password";
      return;
    }

    loading = true;
    error = "";

    try {
      const response = await fetchBackend(`/auth/login`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ username, password }),
      });

      if (response.ok) {
        try {
          const token = await response.json().then(data => data.token);
          if (token) {
            localStorage.setItem(`${API_AUTH_TOKEN}`, token);
          } else {
            console.error("No token received in response");
          }
        } catch (testErr) {
          console.log("Cookie test request failed:", testErr);
        }

        goto("/robots");
      } else {
        if (response.status === 401) {
          error = "Invalid username or password.";
        } else {
          error = "Login failed. Please try again later.";
        }
      }
    } catch (err) {
      console.error("Network error:", err);
      error = "Network error. Please try again.";
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-card">
  <div class="card-header">
    <h1>Sign in</h1>
    <p>Access your robot dashboard</p>
  </div>

  <form on:submit|preventDefault={handleLogin}>
    <div class="form-group">
      <label for="username">Username</label>
      <input
        id="username"
        type="text"
        bind:value={username}
        disabled={loading}
        placeholder="Enter your username"
        required
      />
    </div>

    <div class="form-group">
      <label for="password">Password</label>
      <input
        id="password"
        type="password"
        bind:value={password}
        disabled={loading}
        placeholder="Enter your password"
        required
      />
    </div>

    {#if error}
      <div class="error-msg">{error}</div>
    {/if}

    <button type="submit" disabled={loading} class="login-btn">
      {#if loading}
        <span class="spinner"></span>
        Signing in...
      {:else}
        Sign in
      {/if}
    </button>
  </form>
</div>

<style>
  .login-card {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 2.5rem;
    width: 100%;
    max-width: 400px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  }

  .card-header {
    margin-bottom: 2rem;
  }

  .card-header h1 {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
    margin: 0 0 0.25rem 0;
    letter-spacing: -0.02em;
  }

  .card-header p {
    color: var(--text-secondary);
    margin: 0;
    font-size: 0.9rem;
  }

  .form-group {
    margin-bottom: 1.25rem;
  }

  label {
    display: block;
    margin-bottom: 0.4rem;
    font-weight: 500;
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  input {
    width: 100%;
    padding: 0.7rem 0.9rem;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    font-size: 0.95rem;
    color: var(--text-primary);
    transition: border-color 0.15s, box-shadow 0.15s;
    outline: none;
  }

  input::placeholder {
    color: var(--text-muted);
  }

  input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-muted);
  }

  input:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .login-btn {
    width: 100%;
    padding: 0.75rem;
    margin-top: 0.5rem;
    background: var(--accent);
    color: #0b1120;
    border: none;
    border-radius: var(--radius-sm);
    font-size: 0.95rem;
    font-weight: 600;
    cursor: pointer;
    transition: background-color 0.15s;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
  }

  .login-btn:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .login-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .error-msg {
    background: var(--error-muted);
    color: var(--error);
    padding: 0.65rem 0.9rem;
    border-radius: var(--radius-sm);
    margin-bottom: 1rem;
    font-size: 0.875rem;
    border: 1px solid rgba(239, 68, 68, 0.2);
  }

  .spinner {
    width: 16px;
    height: 16px;
    border: 2px solid rgba(11, 17, 32, 0.3);
    border-top-color: #0b1120;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
