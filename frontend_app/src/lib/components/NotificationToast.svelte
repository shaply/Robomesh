<script lang="ts">
    import { notifications, removeNotification, type Notification } from '$lib/stores/notifications.js';
    import { slide } from 'svelte/transition';
    import { quintOut } from 'svelte/easing';

    // Icons for different notification types
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
        gap: 12px;
        max-width: 400px;
        pointer-events: none;
    }

    .notification {
        background: white;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        display: flex;
        align-items: flex-start;
        gap: 12px;
        padding: 16px;
        border-left: 4px solid;
        pointer-events: auto;
        font-family: inherit;
    }

    .notification--success {
        border-left-color: #10b981;
    }

    .notification--error {
        border-left-color: #ef4444;
    }

    .notification--warning {
        border-left-color: #f59e0b;
    }

    .notification--info {
        border-left-color: #3b82f6;
    }

    .notification__icon {
        width: 20px;
        height: 20px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: bold;
        font-size: 12px;
        color: white;
        flex-shrink: 0;
        margin-top: 2px;
    }

    .notification--success .notification__icon {
        background-color: #10b981;
    }

    .notification--error .notification__icon {
        background-color: #ef4444;
    }

    .notification--warning .notification__icon {
        background-color: #f59e0b;
    }

    .notification--info .notification__icon {
        background-color: #3b82f6;
    }

    .notification__content {
        flex: 1;
        min-width: 0;
    }

    .notification__title {
        font-weight: 600;
        font-size: 14px;
        color: #1f2937;
        margin-bottom: 4px;
    }

    .notification__message {
        font-size: 13px;
        color: #6b7280;
        line-height: 1.4;
    }

    .notification__component-content {
        font-size: 13px;
        color: #374151;
        line-height: 1.4;
    }

    .notification__custom-content {
        font-size: 13px;
        color: #374151;
        line-height: 1.4;
    }

    /* Style elements inside custom content */
    .notification__custom-content :global(button) {
        background: #f3f4f6;
        border: 1px solid #d1d5db;
        border-radius: 4px;
        padding: 4px 8px;
        font-size: 12px;
        font-weight: 500;
        color: #374151;
        cursor: pointer;
        margin-right: 6px;
        margin-top: 6px;
        transition: all 0.2s;
    }

    .notification__custom-content :global(button:hover) {
        background: #e5e7eb;
        border-color: #9ca3af;
    }

    .notification__custom-content :global(button.primary) {
        background: #3b82f6;
        border-color: #3b82f6;
        color: white;
    }

    .notification__custom-content :global(button.primary:hover) {
        background: #2563eb;
        border-color: #2563eb;
    }

    .notification__custom-content :global(button.danger) {
        background: #ef4444;
        border-color: #ef4444;
        color: white;
    }

    .notification__custom-content :global(button.danger:hover) {
        background: #dc2626;
        border-color: #dc2626;
    }

    .notification__custom-content :global(input) {
        border: 1px solid #d1d5db;
        border-radius: 4px;
        padding: 4px 8px;
        font-size: 12px;
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
        gap: 8px;
        flex-shrink: 0;
    }

    .notification__action-btn {
        background: transparent;
        border: 1px solid #d1d5db;
        border-radius: 4px;
        padding: 4px 8px;
        font-size: 12px;
        font-weight: 500;
        color: #374151;
        cursor: pointer;
        transition: all 0.2s;
    }

    .notification__action-btn:hover {
        background: #f3f4f6;
        border-color: #9ca3af;
    }

    .notification__close {
        background: transparent;
        border: none;
        font-size: 16px;
        color: #9ca3af;
        cursor: pointer;
        padding: 0;
        width: 20px;
        height: 20px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 4px;
        transition: all 0.2s;
    }

    .notification__close:hover {
        background: #f3f4f6;
        color: #6b7280;
    }

    /* Mobile responsiveness */
    @media (max-width: 480px) {
        .notification-container {
            left: 20px;
            right: 20px;
            max-width: none;
        }
        
        .notification {
            padding: 12px;
        }
    }
</style>
