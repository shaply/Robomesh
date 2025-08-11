import { writable } from 'svelte/store';

// Define a generic component type that works with Svelte 5
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type SvelteComponent = any;

/**
 * Notification int// Custom notification with HTML content
export function notifyCustom(
    type: 'success' | 'error' | 'warning' | 'info',
    title: string,
    customContent: string,
    duration?: number | null
): string {
    return pushNotification({ 
        type, 
        title, 
        customContent, 
        duration 
    });
}

// Component notification with Svelte component
export function notifyComponent(
    type: 'success' | 'error' | 'warning' | 'info',
    title: string,
    componentClass: ComponentType,
    props?: Record<string, any>,
    duration?: number | null
): string {
    return pushNotification({
        type,
        title,
        component: {
            componentClass,
            props
        },
        duration
    });
}g the structure of a notification object.
 * 
 * Only one of message, customContent, or component should be provided.
 * - message: Simple text message
 * - customContent: Raw HTML content
 * - component: Svelte component with props, the notification id will be passed in along with the props as notificationId
 * 
 * @property {string} message - The message content of the notification.
 * @property {string} customContent - Optional HTML content for custom notifications.
 * @property {object} component - Optional Svelte component configuration.
 */
export interface Notification {
    id: string;
    type: 'success' | 'error' | 'warning' | 'info';
    title: string;
    message?: string;
    customContent?: string; // HTML content for custom notifications
    component?: {
        componentClass: SvelteComponent;
        props?: Record<string, any>;
    };
    duration?: number | null; // in milliseconds, null for persistent
    action?: {
        label: string;
        callback: () => void;
    };
}

export const notifications = writable<Notification[]>([]);

let notificationId = 0;
const timeoutMap = new Map<string, number>(); // Track timeouts to clear them properly

/**
 * 
 * @param notification The notification object to push. It should not include the 'id' field, which will be generated automatically.
 * @returns 
 */
export function pushNotification(notification: Omit<Notification, 'id'>): string {
    const id = `notification-${++notificationId}`;
    const newNotification: Notification = {
        id,
        ...notification
    };
    if (newNotification.duration === undefined || (newNotification.duration !== null && newNotification.duration <= 0)) {
        newNotification.duration = 5000; // Default to 5 seconds if not specified
    }

    notifications.update(n => [...n, newNotification]);

    // Auto-remove notification after duration (if not persistent)
    if (newNotification.duration) {
        const timeoutId = setTimeout(() => {
            removeNotification(id);
        }, newNotification.duration);
        
        // Store timeout ID so we can clear it if needed
        timeoutMap.set(id, timeoutId);
    }

    return id;
}

export function removeNotification(id: string): void {
    // Clear the timeout if it exists
    const timeoutId = timeoutMap.get(id);
    if (timeoutId) {
        clearTimeout(timeoutId);
        timeoutMap.delete(id);
    }
    
    notifications.update(n => n.filter(notification => notification.id !== id));
}

export function clearAllNotifications(): void {
    // Clear all timeouts
    timeoutMap.forEach((timeoutId) => clearTimeout(timeoutId));
    timeoutMap.clear();
    
    notifications.set([]);
}

// Convenience functions for common notification types
export function notifySuccess(title: string, message?: string, duration?: number | null): string {
    return pushNotification({ type: 'success', title, message, duration });
}

export function notifyError(title: string, message?: string, duration?: number | null): string {
    return pushNotification({ type: 'error', title, message, duration });
}

/**
 * 
 * @param title The title of the notification
 * @param message The message content of the notification
 * @param duration The duration (ms) for which the notification should be displayed. Pass null for persistent notifications.
 * @returns The ID of the created notification
 */
export function notifyWarning(title: string, message?: string, duration?: number | null): string {
    return pushNotification({ type: 'warning', title, message, duration });
}

export function notifyInfo(title: string, message?: string, duration?: number | null): string {
    return pushNotification({ type: 'info', title, message, duration });
}

// Custom notification with HTML content
export function notifyCustom(
    type: 'success' | 'error' | 'warning' | 'info',
    title: string,
    customContent: string,
    duration?: number | null
): string {
    return pushNotification({ 
        type, 
        title, 
        customContent, 
        duration: duration ?? 5000 
    });
}

// Component notification with Svelte component
export function notifyComponent(
    type: 'success' | 'error' | 'warning' | 'info',
    title: string,
    componentClass: SvelteComponent,
    props?: Record<string, any>,
    duration?: number | null
): string {
    return pushNotification({
        type,
        title,
        component: {
            componentClass,
            props
        },
        duration: duration ?? 5000
    });
}
