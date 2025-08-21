<script lang="ts">
  // Props passed from the parent
  let { robot } = $props<{ robot: any }>();
</script>

<div class="default-robot">
  <h2>Robot: {robot.device_id}</h2>
  <div class="robot-info">
    <p><strong>Type:</strong> {robot.robot_type || 'Unknown'}</p>
    <p><strong>Status:</strong> {robot.status || 'Unknown'}</p>
    <p><strong>Last Seen:</strong> {robot.last_seen || 'Never'}</p>
  </div>
  
  <div class="controls">
    <h3>Basic Controls</h3>
    <button class="control-btn">Start</button>
    <button class="control-btn">Stop</button>
    <button class="control-btn">Reset</button>
  </div>
  
  <div class="info-section">
    <h3>Robot Information</h3>
    <pre>{JSON.stringify(robot, null, 2)}</pre>
  </div>
</div>

<style>
  .default-robot {
    padding: 1rem;
  }
  
  .robot-info {
    background: #f5f5f5;
    padding: 1rem;
    border-radius: 0.5rem;
    margin: 1rem 0;
  }
  
  .robot-info p {
    margin: 0.5rem 0;
  }
  
  .controls {
    margin: 1rem 0;
  }
  
  .control-btn {
    background: #2563eb;
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    margin: 0.25rem;
    border-radius: 0.25rem;
    cursor: pointer;
  }
  
  .control-btn:hover {
    background: #1d4ed8;
  }
  
  .info-section {
    margin-top: 2rem;
  }
  
  .info-section pre {
    background: #f8f9fa;
    padding: 1rem;
    border-radius: 0.25rem;
    overflow-x: auto;
    font-size: 0.875rem;
  }
</style>
