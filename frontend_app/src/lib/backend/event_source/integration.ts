import { notifySuccess, notifyError, notifyInfo, notifyWarning } from '$lib/stores/notifications.js';
import { eventSourceManager } from './EventSourceManager.js';

// Example function that shows how to integrate notifications with your EventSource system
export function connectWithNotifications() {
    try {
        // Show connecting notification
        notifyInfo('Connecting...', 'Establishing connection to event stream');
        
        // Connect to EventSource
        eventSourceManager.connect();
        
        // You could also listen for connection events and show appropriate notifications
        // This is just an example - you'd implement this based on your EventSource events
        
        // Example: Listen for robot status updates and show notifications
        eventSourceManager.subscribe('robot-status', (event) => {
            const { robot_id, status, message } = JSON.parse(event.data);
            
            if (status === 'online') {
                notifySuccess(
                    `Robot ${robot_id} Online`, 
                    `Robot ${robot_id} is now connected and ready`
                );
            } else if (status === 'offline') {
                notifyError(
                    `Robot ${robot_id} Offline`, 
                    `Robot ${robot_id} has disconnected`
                );
            } else if (status === 'error') {
                notifyError(
                    `Robot ${robot_id} Error`, 
                    message || 'An error occurred with this robot'
                );
            }
        });
        
        // Example: Listen for system notifications
        eventSourceManager.subscribe('system-notification', (event) => {
            const { type, title, message } = JSON.parse(event.data);

            switch (type) {
                case 'maintenance':
                    notifyWarning(title, message);
                    break;
                case 'update':
                    notifyInfo(title, message);
                    break;
                case 'alert':
                    notifyError(title, message);
                    break;
                default:
                    notifyInfo(title, message);
            }
        });
        
    } catch (error) {
        notifyError('Connection Failed', 'Unable to connect to event stream');
        console.error('EventSource connection error:', error);
    }
}

// Example function for handling EventSource errors with notifications
export function handleEventSourceError(error: any) {
    notifyError(
        'Connection Lost', 
        'Lost connection to event stream. Attempting to reconnect...'
    );
    
    // You could implement reconnection logic here
    setTimeout(() => {
        connectWithNotifications();
    }, 5000);
}
