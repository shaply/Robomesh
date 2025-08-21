<script lang="ts">
  import { getRobotComponent } from '$lib/robots/registry.js';
  
  // Get the robot data from the layout
  let { data } = $props();
  
  // Get the appropriate component for this robot type
  const robotConfig = $derived(getRobotComponent(data.robot?.robot_type || 'default'));
  const RobotComponent = $derived(robotConfig.component);
</script>

<div class="robot-page">
  {#if data.robot}
    <div class="robot-header">
      <h1>{robotConfig.displayName}</h1>
      {#if robotConfig.description}
        <p class="description">{robotConfig.description}</p>
      {/if}
    </div>
    
    <!-- Dynamically render the robot-specific component -->
    <RobotComponent robot={data.robot} />
  {:else}
    <div class="error">
      <h2>Robot not found</h2>
      <p>The robot you're looking for doesn't exist or couldn't be loaded.</p>
    </div>
  {/if}
</div>

<style>
  .robot-page {
    max-width: 1200px;
    margin: 0 auto;
    padding: 1rem;
  }
  
  .robot-header {
    margin-bottom: 2rem;
    text-align: center;
  }
  
  .robot-header h1 {
    color: #2c516e;
    margin-bottom: 0.5rem;
  }
  
  .description {
    color: #666;
    font-style: italic;
  }
  
  .error {
    text-align: center;
    padding: 2rem;
    background: #fee2e2;
    border: 1px solid #fecaca;
    border-radius: 0.5rem;
    color: #991b1b;
  }
</style>