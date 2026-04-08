import { browser } from '$app/environment';
import { SSE_SESSION_ID_EVENT } from '$lib/const.js';
import { fetchBackend, backendBaseUrl } from '../fetch.js';
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
        let op: 'subscribe' | 'unsubscribe' | null = null;
        let list: string[] = [];

        if (import.meta.env.DEV) console.log('Processing operation queue:', this.operationQueue);

        while (this.operationQueue.length > 0) {
            const operation = this.operationQueue.shift();
            if (!operation) break;

            // Flush the current batch if the operation type changed
            if (op !== null && op !== operation.type) {
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
                    });
                    if (!response.ok) {
                        this.connReset();
                        console.error(`Failed to ${op} events:`, response.statusText, await response.text());
                        throw new Error(`Failed to ${op} events: ${response.statusText}`);
                    }
                    list = [];
                }
            }

            op = operation.type;
            list.push(operation.eventType);
        }

        // Flush remaining batch
        if (op !== null && list.length > 0) {
            const response = await fetchBackend(`/events/${op}`, {
                method: 'POST',
                body: JSON.stringify({
                    event_types: list,
                    event_session: this.eSess
                }),
                headers: {
                    'Content-Type': 'application/json'
                }
            });
            if (!response.ok) {
                this.connReset();
                console.error(`Failed to ${op} events:`, response.statusText, await response.text());
                throw new Error(`Failed to ${op} events: ${response.statusText}`);
            }
        }

        this.isProcessingQueue = false;
    }

    async connect(): Promise<void> {
        if (!browser || this.isConnected() || this.isConnecting) return;

        this.isConnecting = true;

        // Request a short-lived single-use ticket instead of putting the JWT in the URL
        let ticket: string;
        try {
            const ticketRes = await fetchBackend('/auth/ticket', { method: 'POST' });
            if (!ticketRes.ok) {
                console.error('Failed to obtain SSE ticket');
                this.isConnecting = false;
                return;
            }
            const ticketData = await ticketRes.json();
            ticket = ticketData.ticket;
        } catch (e) {
            console.error('Failed to fetch SSE ticket:', e);
            this.isConnecting = false;
            return;
        }

        // Build URL with subscribed events and single-use ticket
        const eventTypes = Array.from(this.eventMap.keys());
        const eventParam = `events=${eventTypes.map(event => encodeURIComponent(event)).join(',')}`;
        const url = `/events?${eventParam}&ticket=${encodeURIComponent(ticket)}`;
        const fullUrl = `${backendBaseUrl()}${url}`;

        if (import.meta.env.DEV) console.log('Connecting EventSource with events:', eventTypes);

        this.eventSource = new EventSource(fullUrl, {
            withCredentials: true
        });
        if (!this.eventSource) {
            console.error('Failed to create EventSource instance');
            this.connReset();
            return;
        }

        this.eventSource.onmessage = (event) => {
            if (import.meta.env.DEV) console.log('Received message:', event);
            this.handleEvent(event);
        };

        this.eventSource.onopen = () => {
            if (import.meta.env.DEV) console.log('EventSource connected');
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
            if (import.meta.env.DEV) console.log('EventSource disconnected');
        }
        this.isConnecting = false;
    }

    isConnected(): boolean {
        return this.eventSource?.readyState === EventSource.OPEN && this.eSess !== null;
    }

    getSubscribedEvents(): string[] {
        return Array.from(this.eventMap.keys());
    }

    // stripHtmlTags removes HTML tags from a string to prevent XSS when data
    // is accidentally rendered as HTML (e.g., via {@html} in Svelte).
    private static stripHtmlTags(str: string): string {
        return str.replace(/<[^>]*>/g, '');
    }

    private handleEvent(event: MessageEvent): void {
        try {
            const sentEvent = JSON.parse(event.data) as SentEvent;

            // Sanitize event data — strip HTML tags to prevent XSS
            const eventData: EventData = {
                type: sentEvent.type,
                data: EventSourceManager.stripHtmlTags(sentEvent.data),
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

            if (import.meta.env.DEV) console.log(`Received ${sentEvent.type}:`, eventData);
        } catch (error) {
            console.error(`Error parsing event:`, error);
        }
    }

    // Method to reconnect with updated subscriptions
    async reconnect(): Promise<void> {
        this.disconnect();
        await this.connect();
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

    private handleEsessEvent(event: EventData): void {
        const data = JSON.parse(event.data);
        this.eSess = data || null;
        if (import.meta.env.DEV) console.log('Received session ID:', this.eSess);

        // Process any queued operations now that we have the session ID
        if (this.queueProcessPromise) {
            this.queueProcessPromise.then(() => this.processQueue());
        } else {
            this.processQueue();
        }
    }
}

// Export singleton instance
export const eventSourceManager = EventSourceManager.getInstance();
