<script>
  /**
   * Template Robot Card component.
   * This is rendered in the main dashboard's robot grid.
   *
   * Props are injected by the main app:
   * @prop {object} robot - The robot data (uuid, ip, device_type, status, etc.)
   */
  let { robot = {} } = $props();
</script>

<div class="robot-card-custom">
  <p>Custom card for {robot.device_type || 'unknown'}</p>
  <p class="uuid">{robot.device_id || robot.uuid || 'N/A'}</p>
</div>

<style>
  .robot-card-custom {
    padding: 1rem;
    border-radius: 0.5rem;
    background: var(--card-bg, #1e293b);
    color: var(--card-text, #e2e8f0);
  }
  .uuid {
    font-family: monospace;
    font-size: 0.75rem;
    opacity: 0.7;
  }
</style>
