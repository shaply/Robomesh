<script lang="ts">
  import { getRobots } from "$lib/backend/get_robots.js";
  import PageButton from "$lib/components/page-button.svelte";
  import RobotCard from "$lib/components/RobotCard.svelte";
  import SearchBar from "$lib/components/search-bar.svelte";
  import type { BaseRobot } from "$lib/types.js";
  import { onMount, onDestroy } from "svelte";

  let robots: BaseRobot[] = [];
  let loading = true;
  let error: string | null = null;
  let intervalId: number;

  async function fetchRobots() {
    try {
      loading = true;
      error = null;
      const response = await getRobots();
      if (!response) {
        throw new Error(`Failed to fetch robots`);
      }
      robots = response;
    } catch (err) {
      console.error("Error fetching robots:", err);
      error = err instanceof Error ? err.message : "Failed to fetch robots";
      robots = [];
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchRobots();
    intervalId = setInterval(fetchRobots, 60000);
  });

  onDestroy(() => {
    if (intervalId) {
      clearInterval(intervalId);
    }
  });

  function refreshRobots() {
    fetchRobots();
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Active Robots</h1>
    <p class="page-subtitle">Currently connected robots and their status</p>
  </div>

  <div class="toolbar">
    <div class="search-area">
      <SearchBar></SearchBar>
    </div>
    <div class="toolbar-actions">
      <PageButton>Search</PageButton>
      <PageButton>Filter+</PageButton>
      <PageButton on:click={refreshRobots} variant="secondary">Refresh</PageButton>
    </div>
  </div>

  <div class="cards-area">
    {#if loading}
      <div class="state-message">
        <span class="spinner"></span>
        Loading robots...
      </div>
    {:else if error}
      <div class="state-message state-error">Error: {error}</div>
    {:else if robots.length === 0}
      <div class="state-message">
        <span class="empty-label">No robots connected</span>
        <span class="empty-hint">Robots will appear here when they connect via TCP</span>
      </div>
    {:else}
      <div class="cards-grid">
        {#each robots as robot (robot.device_id)}
          <RobotCard {robot} />
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .page {
    max-width: 1200px;
  }

  .page-header {
    margin-bottom: 1.75rem;
  }

  .page-header h1 {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
    margin: 0 0 0.25rem 0;
    letter-spacing: -0.02em;
  }

  .page-subtitle {
    color: var(--text-secondary);
    font-size: 0.9rem;
    margin: 0;
  }

  .toolbar {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1.5rem;
  }

  .search-area {
    flex: 1;
    max-width: 400px;
  }

  .toolbar-actions {
    display: flex;
    align-items: center;
    gap: 0.35rem;
  }

  .cards-area {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 1.25rem;
    min-height: 300px;
  }

  .cards-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1rem;
  }

  .state-message {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 3rem 1rem;
    color: var(--text-secondary);
    font-size: 0.95rem;
  }

  .state-error {
    color: var(--error);
  }

  .empty-label {
    font-weight: 600;
    color: var(--text-secondary);
  }

  .empty-hint {
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .spinner {
    width: 20px;
    height: 20px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  @media (max-width: 768px) {
    .toolbar {
      flex-direction: column;
      align-items: stretch;
    }

    .search-area {
      max-width: none;
    }

    .toolbar-actions {
      justify-content: flex-end;
    }

    .cards-grid {
      grid-template-columns: 1fr;
    }

    .cards-area {
      padding: 0.75rem;
    }
  }
</style>
