<script>
  let { children } = $props();
</script>

<main class="auth-layout">
  <nav class="auth-nav">
    <a href="/login" class="auth-logo">
      <span class="logo-icon">R</span>
      <span class="logo-text">Robomesh</span>
    </a>
  </nav>
  <div class="auth-content">
    {@render children()}
  </div>
  <footer class="auth-footer">
    <span>Robomesh &mdash; Autonomous Robot Management</span>
  </footer>
</main>

<style>
  .auth-layout {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    background:
      radial-gradient(ellipse at 20% 50%, rgba(56, 189, 248, 0.06) 0%, transparent 50%),
      radial-gradient(ellipse at 80% 20%, rgba(56, 189, 248, 0.04) 0%, transparent 50%),
      var(--bg-base);
  }

  .auth-nav {
    padding: 1.5rem 2rem;
  }

  .auth-logo {
    display: inline-flex;
    align-items: center;
    gap: 0.75rem;
    text-decoration: none;
  }

  .logo-icon {
    width: 36px;
    height: 36px;
    background: linear-gradient(135deg, var(--accent), #0284c7);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: 1.1rem;
    color: white;
  }

  .logo-text {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.02em;
  }

  .auth-content {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 2rem;
  }

  .auth-footer {
    padding: 1.5rem 2rem;
    text-align: center;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
</style>
