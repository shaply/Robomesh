<script lang="ts">
  import { getRobots } from "$lib/backend/get_robots.js";
  import PageButton from "$lib/components/page-button.svelte";
  import RobotCard from "$lib/components/RobotCard.svelte";
  import SearchBar from "$lib/components/search-bar.svelte";
  import type { BaseRobot } from "$lib/types.js";
  import { onMount, onDestroy } from "svelte";

  let robots = $state<BaseRobot[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let intervalId: number;

  let searchQuery = $state("");
  let filterType = $state("");
  let showFilters = $state(false);

  let deviceTypes = $derived([...new Set(robots.map((r) => r.robot_type))].sort());

  let filteredRobots = $derived(robots.filter((robot) => {
    const matchesSearch =
      searchQuery === "" ||
      robot.device_id.toLowerCase().includes(searchQuery.toLowerCase()) ||
      robot.robot_type.toLowerCase().includes(searchQuery.toLowerCase()) ||
      robot.ip?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      robot.name?.toLowerCase().includes(searchQuery.toLowerCase());

    const matchesType = filterType === "" || robot.robot_type === filterType;

    return matchesSearch && matchesType;
  }));

  async function fetchRobots(showLoading = true) {
    try {
      if (showLoading) loading = true;
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
    intervalId = setInterval(() => fetchRobots(false), 60000);
  });

  onDestroy(() => {
    if (intervalId) {
      clearInterval(intervalId);
    }
  });

  function refreshRobots() {
    fetchRobots();
  }

  function clearFilters() {
    searchQuery = "";
    filterType = "";
    showFilters = false;
  }
</script>

<div class="page">
  <div class="page-header">
    <h1>Active Robots</h1>
    <p class="page-subtitle">Currently connected robots and their status</p>
  </div>

  <div class="toolbar">
    <div class="search-area">
      <SearchBar
        placeholder="Search by name, UUID, IP, or type..."
        onChange={(value) => (searchQuery = value)}
      ></SearchBar>
    </div>
    <div class="toolbar-actions">
      <PageButton onclick={() => (showFilters = !showFilters)}>
        {showFilters ? "Hide Filters" : "Filter+"}
      </PageButton>
      {#if searchQuery || filterType}
        <PageButton onclick={clearFilters} variant="secondary">Clear</PageButton>
      {/if}
      <PageButton onclick={refreshRobots} variant="secondary">Refresh</PageButton>
    </div>
  </div>

  {#if showFilters}
    <div class="filter-bar">
      <label class="filter-group">
        <span class="filter-label">Device Type</span>
        <select class="filter-select" bind:value={filterType}>
          <option value="">All Types</option>
          {#each deviceTypes as type}
            <option value={type}>{type}</option>
          {/each}
        </select>
      </label>
    </div>
  {/if}

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
    {:else if filteredRobots.length === 0}
      <div class="state-message">
        <span class="empty-label">No matching robots</span>
        <span class="empty-hint">Try adjusting your search or filter criteria</span>
      </div>
    {:else}
      {#if searchQuery || filterType}
        <div class="result-count">
          Showing {filteredRobots.length} of {robots.length} robots
        </div>
      {/if}
      <div class="cards-grid">
        {#each filteredRobots as robot (robot.device_id)}
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
    margin-bottom: 1rem;
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

  .filter-bar {
    display: flex;
    gap: 1rem;
    margin-bottom: 1rem;
    padding: 0.75rem 1rem;
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
  }

  .filter-group {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .filter-label {
    font-size: 0.82rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .filter-select {
    padding: 0.35rem 0.65rem;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-size: 0.85rem;
    outline: none;
  }

  .filter-select:focus {
    border-color: var(--accent);
  }

  .result-count {
    font-size: 0.82rem;
    color: var(--text-muted);
    margin-bottom: 0.75rem;
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

    .filter-bar {
      flex-direction: column;
    }
  }
</style>
