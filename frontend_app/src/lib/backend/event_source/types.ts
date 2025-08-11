export type EventHandler = (eventData: EventData) => void;

export interface EventData {
    type: string;
    data: string;
    timestamp: number;
    id?: string; // SSE event ID
}

export type EventMap = Map<string, EventHandler[]>;

export interface SentEvent {
    id: string;
    type: string;
    encoded_data: string;
}