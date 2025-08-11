<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { browser } from '$app/environment';
  import NotificationToast from '$lib/components/NotificationToast.svelte';
  import { notifySuccess, notifyWarning, notifyError, removeNotification, pushNotification } from '$lib/index.js';
  import { eventSourceManager } from '$lib/backend/event_source/EventSourceManager.js';
  import type { EventData } from '$lib/backend/event_source/types.js';
  import { fetchBackend } from '$lib/backend/fetch.js';
  import { API_AUTH_TOKEN } from '$lib/const.js';
  import { authFailureRedirect } from '$lib/utils/failure_redirect.js';
  import type { RegisteringRobotEvent } from '$lib/types.js';
  import RegisteringRobotNotifComponent from '$lib/components/RegisteringRobotNotifComponent.svelte';
  
  let { children } = $props();

  function notify_registering_robot(event: EventData) {
    const data = JSON.parse(event.data) as RegisteringRobotEvent;
    console.log("Registering robot:", data);
    const notId = pushNotification({
      type: 'info',
      title: 'Registering Robot',
      component: {
        componentClass: RegisteringRobotNotifComponent,
        props: {
          registering_robot: data
        }
      },
      duration: null
    });
  }

  onMount(async () => {
    if (!browser) return;

    const authToken = localStorage.getItem(`${API_AUTH_TOKEN}`);
    if (!authToken) {
      authFailureRedirect();
      return;
    }

    let warningNotificationId: string | null = null;
    let isResolved = false;

    // Show warning after 3 seconds if still pending
    const warningTimeout = setTimeout(() => {
      if (!isResolved) {
        warningNotificationId = notifyWarning(
          "Slow Connection", 
          "Connecting to server is taking longer than usual...",
          10000
        );
      }
    }, 3000);

    try {
      const response = await fetchBackend("/auth", {
        method: "GET",
      });

      isResolved = true;
      clearTimeout(warningTimeout);

      // Remove warning notification if it was shown
      if (warningNotificationId) {
        removeNotification(warningNotificationId);
      }

      if (response.ok) {
        notifySuccess("Connected", "Successfully connected to server");
        eventSourceManager.subscribe("robot_manager.registering_robot", notify_registering_robot);
      } else if (response.statusText === "Network error") {
        console.error("Network error while fetching user data");
        notifyError("Network Error", "Please check your internet connection.");
      } else {
        console.error("Failed to fetch user data:", response.statusText);
        authFailureRedirect();
      }
    } catch (error) {
      isResolved = true;
      clearTimeout(warningTimeout);

      // Remove warning notification if it was shown
      if (warningNotificationId) {
        removeNotification(warningNotificationId);
      }

      console.error("Auth request failed:", error);
      notifyError("Connection Failed", "Unable to connect to server. Please try refreshing.");
      // Don't redirect immediately - let user try to refresh or wait
    }

    // Set up EventSource for real-time updates
    eventSourceManager.connect();
  });

  onDestroy(async () => {
    eventSourceManager.disconnect();
  })
</script>

<main class="app-layout">
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

<!-- Global notification system -->
<NotificationToast />

<style>
  .app-layout {
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

  .content {
    padding: 1.5rem 2rem 2rem 2rem;
  }
</style>
