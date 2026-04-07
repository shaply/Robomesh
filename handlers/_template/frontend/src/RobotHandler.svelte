<script>
  /**
   * Template Robot Handler Page component.
   * This is rendered as a full-page SPA when the user clicks into a robot's detail view.
   *
   * Props are injected by the main app:
   * @prop {object} robot - The robot data
   * @prop {function} fetchBackend - Function to make authenticated API calls
   * @prop {function} sendToHandler - Function to send a message to the handler process
   */
  let { robot = {}, fetchBackend = null, sendToHandler = null } = $props();

  let commandInput = $state('');
  let logs = $state([]);

  function sendCommand() {
    if (!commandInput.trim() || !sendToHandler) return;
    logs = [...logs, { type: 'sent', text: commandInput }];
    sendToHandler(commandInput);
    commandInput = '';
  }
</script>

<div class="handler-page">
  <h2>Handler: {robot.device_type || 'Unknown'}</h2>
  <p>UUID: <code>{robot.device_id || robot.uuid || 'N/A'}</code></p>

  <div class="controls">
    <input
      type="text"
      bind:value={commandInput}
      placeholder="Send command to handler..."
      onkeydown={(e) => e.key === 'Enter' && sendCommand()}
    />
    <button onclick={sendCommand}>Send</button>
  </div>

  <div class="logs">
    {#each logs as log}
      <div class="log-entry {log.type}">
        <span class="prefix">{log.type === 'sent' ? '>' : '<'}</span>
        {log.text}
      </div>
    {/each}
  </div>
</div>

<style>
  .handler-page {
    padding: 1.5rem;
    color: var(--card-text, #e2e8f0);
  }
  h2 { margin-bottom: 0.5rem; }
  code { font-family: monospace; opacity: 0.7; }
  .controls {
    display: flex;
    gap: 0.5rem;
    margin: 1rem 0;
  }
  input {
    flex: 1;
    padding: 0.5rem;
    border-radius: 0.25rem;
    border: 1px solid #475569;
    background: #0f172a;
    color: #e2e8f0;
  }
  button {
    padding: 0.5rem 1rem;
    border-radius: 0.25rem;
    background: #38bdf8;
    color: #0f172a;
    border: none;
    cursor: pointer;
  }
  .logs {
    font-family: monospace;
    font-size: 0.875rem;
    max-height: 400px;
    overflow-y: auto;
  }
  .log-entry { padding: 0.25rem 0; }
  .prefix { opacity: 0.5; margin-right: 0.5rem; }
</style>
