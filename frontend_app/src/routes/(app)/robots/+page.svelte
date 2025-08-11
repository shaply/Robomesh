<script lang="ts">
  import { getRobots } from "$lib/backend/get_robots.js";
  import PageButton from "$lib/components/page-button.svelte";
  import RobotCard from "$lib/components/RobotCard.svelte";
  import SearchBar from "$lib/components/search-bar.svelte";
  import type { BaseRobot } from "$lib/types.js";
  import { onMount, onDestroy } from "svelte";
  import { eventSourceManager } from "$lib/backend/event_source/EventSourceManager.js";

  // Receive data from layout (including events array)

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
    // Fetch immediately on mount
    fetchRobots();

    // Set up polling every minute (60000ms)
    intervalId = setInterval(fetchRobots, 60000);
  });

  onDestroy(() => {
    // Clean up interval when component is destroyed
    if (intervalId) {
      clearInterval(intervalId);
    }
  });

  // Function to manually refresh
  function refreshRobots() {
    fetchRobots();
  }
</script>

<main class="main-layout">
  <div class="container-buttons">
    <div class="search-section">
      <SearchBar></SearchBar>
    </div>
    <div class="buttons-section">
      <div class="action-buttons">
        <PageButton>Search</PageButton>
        <PageButton>Filter+</PageButton>
      </div>
      <div class="refresh-buttons">
        <PageButton on:click={refreshRobots}>Refresh Robots</PageButton>
      </div>
    </div>
  </div>
  <div class="robot-cards-container">
    {#if loading}
      <div class="robot-cards-container-text">Loading robots...</div>
    {:else if error}
      <div class="error robot-cards-container-text">Error: {error}</div>
    {:else if robots.length === 0}
      <div class="robot-cards-container-text">No robots found.</div>
    {:else}
      {#each robots as robot (robot.device_id)}
        <RobotCard {robot} />
      {/each}
    {/if}
  </div>
</main>

<style>
  .main-layout {
    font-family: "Roboto", sans-serif;
    background-color: #ddebf2;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .container-buttons {
    display: flex;
    align-items: center;
    margin-bottom: 1rem;
    border: 1px solid transparent;
    gap: 1rem;
  }

  .search-section {
    flex: 1;
    display: flex;
    align-items: center;
  }

  .buttons-section {
    flex: 1;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .action-buttons {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .refresh-buttons {
    display: flex;
    align-items: center;
  }

  .robot-cards-container {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 0.5rem;
    border: 1px solid black;
    flex: 1;
    border-radius: 10px;
    padding: 0.5rem;
    margin: 0.5rem;
    overflow-y: auto;
    background-color: #cedde3;
    align-items: start;
  }

  .robot-cards-container-text {
    grid-column: 1 / -1;
    display: flex;
    justify-content: center;
    padding: 1rem;
    text-align: center;
    color: #374151; /* Dark gray */
    font-size: 1.2rem;
    font-weight: 500;
    font-style: italic;
  }
</style>
