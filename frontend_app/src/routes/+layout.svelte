<script>
  let { children } = $props();
</script>

<main class="main-layout">
  <nav class="navbar">
    <div class="container">
      <a href="/robots" class="logo">Dashboard</a>
      <ul class="nav-links">
        <li><a href="/robots">Robots</a></li>
        <li><a href="/settings">Profile</a></li>
        <li><a href="/logout">Logout</a></li>
      </ul>
    </div>
  </nav>
  <div class="content">
    {@render children()}
  </div>
</main>

<style>
  :global(html, body) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(*, *::before, *::after) {
    box-sizing: inherit;
  }

  .main-layout {
    font-family: 'Roboto', sans-serif;
    background-color: #DDEBF2;
    min-height: 100vh;
  }

  .navbar {
    background-color: #2C516E; /* Dark gray */
    color: white;
    padding-top: 1.5rem;
    padding-bottom: 1.5rem;
    padding-left: 2.5rem;
    padding-right: 2.5rem;
  }

  .content {
    padding-top: 1.5rem;
    padding-left: 2rem;
    padding-right: 2rem;
    padding-bottom: 2rem;
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
    color: white;
  }
  .navbar .nav-links li a {
    color: white;
    text-decoration: none;
  }
  .navbar .nav-links li a:hover {
    text-decoration: underline;
  }
</style>

