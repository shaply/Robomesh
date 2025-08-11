<script>
  let { children } = $props();
</script>

<!-- Add a div wrapper to make styles scopable -->
<div class="app">
  {@render children()}
</div>

<style>
  :global(html, body) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(*, *::before, *::after) {
    box-sizing: inherit;
  }
</style>

