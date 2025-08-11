import { browser } from '$app/environment';
import { PUBLIC_BACKEND_IP, PUBLIC_BACKEND_PORT } from '$env/static/public';
import { API_AUTH_TOKEN, SSE_SESSION_ID_EVENT } from '$lib/const.js';
import { fetchBackend } from '../fetch.js';
import type { EventMap, EventHandler, EventData, SentEvent } from './types.js';

interface QueuedOperation {
    type: 'subscribe' | 'unsubscribe';
    eventType: string;
    timestamp: number;
}

class EventSourceManager {
    private static instance: EventSourceManager | null = null;
    private eventSource: EventSource | null = null;
    private eventMap: EventMap = new Map();
    private eSess: string | null = null;
    private isConnecting = false;
    private isInBackoffMode = false; // Flag to track backoff mode
    
    // Queue management
    private operationQueue: QueuedOperation[] = [];
    private queueProcessPromise: Promise<void> = Promise.resolve();
    private isProcessingQueue = false;
    
    private constructor() {
        // Private constructor for singleton
        if (browser) {
            window.addEventListener('beforeunload', () => this.disconnect());
        }
    }

    static getInstance(): EventSourceManager {
        if (!EventSourceManager.instance) {
            EventSourceManager.instance = new EventSourceManager();
        }
        return EventSourceManager.instance;
    }

    subscribe(eventType: string, handler: EventHandler): Promise<void> {
        const isIn = this.eventMap.has(eventType);
        this.eventMap.set(eventType, [...(this.eventMap.get(eventType) || []), handler]);

        if (!isIn) {
            this.operationQueue.push({
                type: 'subscribe',
                eventType,
                timestamp: Date.now()
            });

            if (this.isConnected()) {
                return this.processQueue();
            }
        }

        return Promise.resolve();
    }

    unsubscribe(eventType: string, handler?: EventHandler): Promise<void> {
        const isIn = this.eventMap.has(eventType);
        if (!isIn) {
            return Promise.resolve();
        }

        let deleted = false;
        if (handler) {
            const handlers = this.eventMap.get(eventType) || [];
            this.eventMap.set(eventType, handlers.filter(h => h !== handler));
            if (this.eventMap.get(eventType)?.length === 0) {
                this.eventMap.delete(eventType);
                deleted = true;
            }
        } else {
            this.eventMap.delete(eventType);
            deleted = true;
        }

        if (deleted) {
            this.operationQueue.push({
                type: 'unsubscribe',
                eventType,
                timestamp: Date.now()
            });

            if (this.isConnected()) {
                return this.processQueue();
            }
        }
        return Promise.resolve();
    };

    private processQueue() {
        if (this.isProcessingQueue) {
            return this.queueProcessPromise;
        }

        this.isProcessingQueue = true;
        this.queueProcessPromise = this._processQueue();
        return this.queueProcessPromise;
    }

    private async _processQueue() {
        let op: 'subscribe' | 'unsubscribe' = 'subscribe';
        let list: string[] = [];

        console.log('Processing operation queue:', this.operationQueue);

        while (this.operationQueue.length >= 0) {
            const operation = this.operationQueue.shift();
            if (!operation || op !== operation.type) {
                if (list.length > 0) {
                    const response = await fetchBackend(`/events/${op}`, {
                        method: 'POST',
                        body: JSON.stringify({ 
                            event_types: list,
                            event_session: this.eSess
                        }),
                        headers: {
                            'Content-Type': 'application/json'
                        }
                    })
                    if (!response.ok) {
                        this.connReset();
                        console.error(`Failed to ${op} events:`, response.statusText, await response.text());
                        throw new Error(`Failed to ${op} events: ${response.statusText}`);
                    }
                }
    

                op = op === 'subscribe' ? 'unsubscribe' : 'subscribe';
            }

            if (!operation) break;
            list.push(operation.eventType);
        }
        this.isProcessingQueue = false;
    }

    connect(): void {
        if (!browser || this.isConnected() || this.isConnecting) return;

        this.isConnecting = true;

        // Get auth token
        const authToken = localStorage.getItem(`${API_AUTH_TOKEN}`);
        if (!authToken) {
            console.error('No auth token found for EventSource');
            this.isConnecting = false;
            return;
        }

        // Build URL with subscribed events
        const eventTypes = Array.from(this.eventMap.keys());
        const eventParam = `events=${eventTypes.map(event => encodeURIComponent(event)).join(',')}`;
        const tokenParam = `${API_AUTH_TOKEN}=${encodeURIComponent(authToken)}`;
        const url = `/events?${eventParam}&${tokenParam}`;
        const fullUrl = `http://${PUBLIC_BACKEND_IP}:${PUBLIC_BACKEND_PORT}${url}`;

        console.log('Connecting EventSource with events:', eventTypes);

        this.eventSource = new EventSource(fullUrl, {
            withCredentials: true
        });
        if (!this.eventSource) {
            console.error('Failed to create EventSource instance');
            this.connReset();
            return;
        }

        this.eventSource.onmessage = (event) => {
            console.log('Received message:', event);

            this.handleEvent(event);
        };

        this.eventSource.onopen = () => {
            console.log('EventSource connected');
            this.isConnecting = false;
        };

        this.eventSource.onerror = (error) => {
            console.error('EventSource error:', error);
            this.isConnecting = false;

            if (!this.isInBackoffMode) {
                this.scheduleReconnect();
            }
        };

    }

    disconnect(): void {
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
            console.log('EventSource disconnected');
        }
        this.isConnecting = false;
    }

    isConnected(): boolean {
        return this.eventSource?.readyState === EventSource.OPEN && this.eSess !== null;
    }

    getSubscribedEvents(): string[] {
        return Array.from(this.eventMap.keys());
    }

    private handleEvent(event: MessageEvent): void {
        try {
            let decodedData: string;
            try {
                // Try base64 decode first
                decodedData = atob(event.data);
            } catch {
                // Fall back to direct JSON parse
                decodedData = event.data;
            }
            const sentEvent = JSON.parse(decodedData) as SentEvent;

            const eventData: EventData = {
                type: sentEvent.type,
                data: atob(sentEvent.encoded_data),
                timestamp: Date.now(),
                id: sentEvent.id
            };

            if (eventData.type === SSE_SESSION_ID_EVENT) {
                this.handleEsessEvent(eventData);
                return; // Skip further processing for session ID events
            }

            // Call all handlers for this event type
            const handlers = this.eventMap.get(sentEvent.type) || [];
            handlers.forEach(handler => {
                try {
                    handler(eventData);
                } catch (handlerError) {
                    console.error(`Error in event handler for ${sentEvent.type}:`, handlerError);
                }
            });

            console.log(`Received ${sentEvent.type}:`, eventData);
        } catch (error) {
            console.error(`Error parsing event:`, error);
        }
    }

    // Method to reconnect with updated subscriptions
    reconnect(): void {
        this.disconnect();
        this.connect();
    }

    private connReset(): void {
        this.disconnect();
        this.eSess = null;
        this.isConnecting = false;
        this.operationQueue = [];
        this.queueProcessPromise = Promise.resolve();
        this.isProcessingQueue = false;
    }

    private scheduleReconnect(): void {
        if (this.isInBackoffMode) return;

        this.isInBackoffMode = true;
        console.warn('Scheduling reconnect in 5 seconds due to error...');

        setTimeout(() => {
            this.isInBackoffMode = false;
            this.reconnect();
        }, 5000);
    }

    // TODO: Change the way the eSess is received
    private handleEsessEvent(event: EventData): void {
        const data = JSON.parse(event.data);
        this.eSess = data || null;
        console.log('Received session ID:', this.eSess);

        // Process any queued operations now that we have the session ID
        if (this.queueProcessPromise) {
            this.queueProcessPromise.then(() => this.processQueue());
        } else {
            this.processQueue();
        }
    }

    print_subscription(eventType: string): void {
        console.log(`Subscribed to event: ${this.eventMap.get(eventType)?.length || 0} handlers for ${eventType}`);
    }
}

// Export singleton instance
export const eventSourceManager = EventSourceManager.getInstance();
