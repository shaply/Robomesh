<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { browser } from '$app/environment';
  import { page } from '$app/stores';
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
  let warningTimeoutId: ReturnType<typeof setTimeout> | null = null;

  const navLinks = [
    { href: '/robots', label: 'Robots' },
    { href: '/provision', label: 'Provision' },
    { href: '/settings', label: 'Settings' },
  ];

  function isActive(href: string, pathname: string): boolean {
    return pathname.startsWith(href);
  }

  function notify_registering_robot(event: EventData) {
    const data = JSON.parse(event.data) as RegisteringRobotEvent;
    pushNotification({
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

    warningTimeoutId = setTimeout(() => {
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
      if (warningTimeoutId) clearTimeout(warningTimeoutId);

      if (warningNotificationId) {
        removeNotification(warningNotificationId);
      }

      if (response.ok) {
        notifySuccess("Connected", "Successfully connected to server");
        eventSourceManager.subscribe("robot.registering", notify_registering_robot);
      } else if (response.statusText === "Network error") {
        notifyError("Network Error", "Please check your internet connection.");
      } else {
        authFailureRedirect();
      }
    } catch (error) {
      isResolved = true;
      if (warningTimeoutId) clearTimeout(warningTimeoutId);

      if (warningNotificationId) {
        removeNotification(warningNotificationId);
      }

      notifyError("Connection Failed", "Unable to connect to server. Please try refreshing.");
    }

    eventSourceManager.connect();
  });

  onDestroy(async () => {
    if (warningTimeoutId) clearTimeout(warningTimeoutId);
    eventSourceManager.unsubscribe("robot.registering", notify_registering_robot);
    eventSourceManager.disconnect();
  })
</script>

<div class="app-shell">
  <nav class="sidebar">
    <a href="/robots" class="sidebar-logo">
      <span class="logo-icon">R</span>
      <span class="logo-text">Robomesh</span>
    </a>

    <div class="sidebar-links">
      {#each navLinks as link}
        <a
          href={link.href}
          class="nav-link"
          class:active={isActive(link.href, $page.url.pathname)}
        >
          {link.label}
        </a>
      {/each}
    </div>

    <div class="sidebar-footer">
      <a href="/logout" class="nav-link logout-link">Logout</a>
    </div>
  </nav>

  <main class="main-content">
    {@render children()}
  </main>
</div>

<NotificationToast />

<style>
  .app-shell {
    display: flex;
    min-height: 100vh;
  }

  /* --- Sidebar --- */
  .sidebar {
    width: 220px;
    background: var(--bg-surface);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    padding: 1.25rem 0.75rem;
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    z-index: 50;
  }

  .sidebar-logo {
    display: flex;
    align-items: center;
    gap: 0.65rem;
    text-decoration: none;
    padding: 0.25rem 0.75rem;
    margin-bottom: 1.75rem;
  }

  .logo-icon {
    width: 32px;
    height: 32px;
    background: linear-gradient(135deg, var(--accent), #0284c7);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: 1rem;
    color: white;
    flex-shrink: 0;
  }

  .logo-text {
    font-size: 1.1rem;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.02em;
  }

  .sidebar-links {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }

  .nav-link {
    display: block;
    padding: 0.55rem 0.75rem;
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 500;
    transition: background-color 0.12s, color 0.12s;
  }

  .nav-link:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .nav-link.active {
    background: var(--accent-muted);
    color: var(--accent);
  }

  .sidebar-footer {
    border-top: 1px solid var(--border);
    padding-top: 0.75rem;
    margin-top: 0.5rem;
  }

  .logout-link {
    color: var(--text-muted);
  }

  .logout-link:hover {
    color: var(--error);
    background: var(--error-muted);
  }

  /* --- Main content --- */
  .main-content {
    flex: 1;
    margin-left: 220px;
    padding: 2rem 2.5rem;
    min-height: 100vh;
  }

  /* Mobile: collapse sidebar to top bar */
  @media (max-width: 768px) {
    .app-shell {
      flex-direction: column;
    }

    .sidebar {
      position: relative;
      width: 100%;
      flex-direction: row;
      align-items: center;
      padding: 0.75rem 1rem;
      border-right: none;
      border-bottom: 1px solid var(--border);
    }

    .sidebar-logo {
      margin-bottom: 0;
      padding: 0;
    }

    .sidebar-links {
      flex-direction: row;
      gap: 0;
      margin-left: 1.5rem;
    }

    .sidebar-footer {
      border-top: none;
      padding-top: 0;
      margin-top: 0;
      margin-left: auto;
    }

    .main-content {
      margin-left: 0;
      padding: 1.5rem;
    }
  }

  @media (max-width: 480px) {
    .sidebar {
      padding: 0.5rem 0.75rem;
    }

    .sidebar-links {
      margin-left: 0.75rem;
      gap: 0;
    }

    .nav-link {
      padding: 0.4rem 0.5rem;
      font-size: 0.82rem;
    }

    .logo-text {
      display: none;
    }

    .main-content {
      padding: 1rem;
    }
  }
</style>
