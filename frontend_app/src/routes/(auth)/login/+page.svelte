<script lang="ts">
  import { goto } from "$app/navigation";
  import { fetchBackend } from "$lib/backend/fetch.js";
  import { API_AUTH_TOKEN } from "$lib/const.js";

  let username = "";
  let password = "";
  let loading = false;
  let error = "";

  async function handleLogin() {
    console.log("Attempting to log in with:", { username, password });
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
        // Login successful, test if cookie is working
        console.log("Login successful, testing cookie...");
        
        // Make a test request to see if cookie is sent
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
          error = "Invalid username or password. Please try again.";
        } else {
          error = "Login failed. Please try again later.";
        }
      }
    } catch (err) {
      console.error("Network error:", err);
      error = "Network error. Please try again.";
    } finally {
      console.log("Login process completed");
      loading = false;
    }
  }
</script>

<div class="login-container">
  <div class="login-card">
    <h1>Robot Dashboard Login</h1>

    <form on:submit|preventDefault={handleLogin}>
      <div class="form-group">
        <label for="username">Username:</label>
        <input
          id="username"
          type="text"
          bind:value={username}
          disabled={loading}
          required
        />
      </div>

      <div class="form-group">
        <label for="password">Password:</label>
        <input
          id="password"
          type="password"
          bind:value={password}
          disabled={loading}
          required
        />
      </div>

      {#if error}
        <div class="error">{error}</div>
      {/if}

      <button type="submit" disabled={loading} class="login-btn">
        {loading ? "Logging in..." : "Login"}
      </button>
    </form>
  </div>
</div>

<style>
  .login-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 60vh;
  }

  .login-card {
    background: white;
    padding: 2rem;
    border-radius: 12px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    width: 100%;
    max-width: 400px;
  }

  h1 {
    text-align: center;
    margin-bottom: 2rem;
    color: #2c516e;
  }

  .form-group {
    margin-bottom: 1rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 500;
    color: #374151;
  }

  input {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 1rem;
    box-sizing: border-box;
  }

  input:focus {
    outline: none;
    border-color: #2c516e;
    box-shadow: 0 0 0 2px rgba(44, 81, 110, 0.2);
  }

  input:disabled {
    background-color: #f3f4f6;
    cursor: not-allowed;
  }

  .login-btn {
    width: 100%;
    padding: 0.75rem;
    background-color: #2c516e;
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    font-weight: 500;
    cursor: pointer;
    transition: background-color 0.2s ease;
  }

  .login-btn:hover:not(:disabled) {
    background-color: #1e3a52;
  }

  .login-btn:disabled {
    background-color: #9ca3af;
    cursor: not-allowed;
  }

  .error {
    background-color: #fee2e2;
    color: #dc2626;
    padding: 0.75rem;
    border-radius: 6px;
    margin-bottom: 1rem;
    text-align: center;
  }
</style>
