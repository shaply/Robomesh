<script lang="ts">
    import { notifications, removeNotification, type Notification } from '$lib/stores/notifications.js';
    import { slide } from 'svelte/transition';
    import { quintOut } from 'svelte/easing';

    const icons = {
        success: '✓',
        error: '✕',
        warning: '⚠',
        info: 'ℹ'
    };

    function handleRemove(id: string) {
        removeNotification(id);
    }

    function handleAction(notification: Notification) {
        if (notification.action) {
            notification.action.callback();
            removeNotification(notification.id);
        }
    }
</script>

<div class="notification-container">
    {#each $notifications as notification (notification.id)}
        <div
            class="notification notification--{notification.type}"
            transition:slide={{ duration: 300, easing: quintOut }}
        >
            <div class="notification__icon">
                {icons[notification.type]}
            </div>

            <div class="notification__content">
                <div class="notification__title">
                    {notification.title}
                </div>
                {#if notification.component}
                    <div class="notification__component-content">
                        <svelte:component
                            this={notification.component.componentClass}
                            {...(notification.component.props || {})}
                            notificationId={notification.id}
                        />
                    </div>
                {:else if notification.customContent}
                    <div class="notification__custom-content">
                        {@html notification.customContent}
                    </div>
                {:else if notification.message}
                    <div class="notification__message">
                        {notification.message}
                    </div>
                {/if}
            </div>

            <div class="notification__actions">
                {#if notification.action}
                    <button
                        class="notification__action-btn"
                        on:click={() => handleAction(notification)}
                    >
                        {notification.action.label}
                    </button>
                {/if}

                <button
                    class="notification__close"
                    on:click={() => handleRemove(notification.id)}
                    aria-label="Close notification"
                >
                    ✕
                </button>
            </div>
        </div>
    {/each}
</div>

<style>
    .notification-container {
        position: fixed;
        top: 20px;
        right: 20px;
        z-index: 1000;
        display: flex;
        flex-direction: column;
        gap: 10px;
        max-width: 400px;
        pointer-events: none;
    }

    .notification {
        background: var(--bg-elevated);
        border: 1px solid var(--border);
        border-radius: var(--radius);
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
        display: flex;
        align-items: flex-start;
        gap: 10px;
        padding: 14px;
        border-left: 3px solid;
        pointer-events: auto;
        font-family: inherit;
    }

    .notification--success { border-left-color: var(--success); }
    .notification--error   { border-left-color: var(--error); }
    .notification--warning { border-left-color: var(--warning); }
    .notification--info    { border-left-color: var(--info); }

    .notification__icon {
        width: 20px;
        height: 20px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: bold;
        font-size: 11px;
        color: white;
        flex-shrink: 0;
        margin-top: 1px;
    }

    .notification--success .notification__icon { background: var(--success); }
    .notification--error .notification__icon   { background: var(--error); }
    .notification--warning .notification__icon { background: var(--warning); }
    .notification--info .notification__icon    { background: var(--info); }

    .notification__content {
        flex: 1;
        min-width: 0;
    }

    .notification__title {
        font-weight: 600;
        font-size: 0.85rem;
        color: var(--text-primary);
        margin-bottom: 3px;
    }

    .notification__message {
        font-size: 0.8rem;
        color: var(--text-secondary);
        line-height: 1.4;
    }

    .notification__component-content {
        font-size: 0.8rem;
        color: var(--text-secondary);
        line-height: 1.4;
    }

    .notification__custom-content {
        font-size: 0.8rem;
        color: var(--text-secondary);
        line-height: 1.4;
    }

    .notification__custom-content :global(button) {
        background: var(--bg-hover);
        border: 1px solid var(--border);
        border-radius: 4px;
        padding: 4px 8px;
        font-size: 12px;
        font-weight: 500;
        color: var(--text-secondary);
        cursor: pointer;
        margin-right: 6px;
        margin-top: 6px;
        transition: all 0.15s;
    }

    .notification__custom-content :global(button:hover) {
        background: var(--border);
        color: var(--text-primary);
    }

    .notification__custom-content :global(button.primary) {
        background: var(--accent);
        border-color: var(--accent);
        color: #0b1120;
    }

    .notification__custom-content :global(button.primary:hover) {
        background: var(--accent-hover);
    }

    .notification__custom-content :global(button.danger) {
        background: var(--error);
        border-color: var(--error);
        color: white;
    }

    .notification__custom-content :global(button.danger:hover) {
        background: #dc2626;
    }

    .notification__custom-content :global(input) {
        background: var(--bg-elevated);
        border: 1px solid var(--border);
        border-radius: 4px;
        padding: 4px 8px;
        font-size: 12px;
        color: var(--text-primary);
        margin-right: 6px;
        margin-top: 4px;
    }

    .notification__custom-content :global(p) {
        margin: 4px 0;
    }

    .notification__custom-content :global(div) {
        margin: 2px 0;
    }

    .notification__actions {
        display: flex;
        align-items: flex-start;
        gap: 6px;
        flex-shrink: 0;
    }

    .notification__action-btn {
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 4px;
        padding: 4px 8px;
        font-size: 12px;
        font-weight: 500;
        color: var(--text-secondary);
        cursor: pointer;
        transition: all 0.15s;
    }

    .notification__action-btn:hover {
        background: var(--bg-hover);
        color: var(--text-primary);
    }

    .notification__close {
        background: transparent;
        border: none;
        font-size: 14px;
        color: var(--text-muted);
        cursor: pointer;
        padding: 0;
        width: 20px;
        height: 20px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 4px;
        transition: all 0.15s;
    }

    .notification__close:hover {
        background: var(--bg-hover);
        color: var(--text-secondary);
    }

    @media (max-width: 480px) {
        .notification-container {
            left: 12px;
            right: 12px;
            max-width: none;
        }

        .notification {
            padding: 12px;
        }
    }
</style>
