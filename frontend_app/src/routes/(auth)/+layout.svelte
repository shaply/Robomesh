<script>
  let { children } = $props();
</script>

<main class="auth-layout">
  <nav class="navbar">
    <div class="container">
      <a href="/login" class="logo">Robot Dashboard</a>
      <ul class="nav-links">
        <li><a href="/login">Login</a></li>
      </ul>
    </div>
  </nav>
  <div class="auth-content">
    {@render children()}
  </div>
</main>

<style>
  .auth-layout {
    font-family: 'Roboto', sans-serif;
    background-color: #DDEBF2;
    min-height: 100vh;
  }

  .navbar {
    background-color: #2C516E;
    color: white;
    padding: 1.5rem 2.5rem;
  }

  .navbar .container {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .navbar .logo {
    font-size: 1.5rem;
    font-weight: bold;
    text-decoration: none;
    color: white;
  }

  .navbar .nav-links {
    list-style: none;
    display: flex;
    gap: 1rem;
    margin: 0;
    padding: 0;
  }

  .navbar .nav-links li a {
    color: white;
    text-decoration: none;
  }

  .navbar .nav-links li a:hover {
    text-decoration: underline;
  }

  .auth-content {
    padding: 2rem;
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: calc(100vh - 120px);
  }
</style>
