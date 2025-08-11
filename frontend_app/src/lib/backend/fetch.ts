import { PUBLIC_BACKEND_IP, PUBLIC_BACKEND_PORT } from "$env/static/public";
import { API_AUTH_TOKEN } from "$lib/const.js";

export async function fetchBackend(url: string, options: RequestInit = {}): Promise<Response> {
    if (!PUBLIC_BACKEND_IP || !PUBLIC_BACKEND_PORT) {
        throw new Error('Backend IP or port is not defined in environment variables');
    }

    const fullUrl = `http://${PUBLIC_BACKEND_IP}:${PUBLIC_BACKEND_PORT}${url}`;

    const token = localStorage.getItem(`${API_AUTH_TOKEN}`);
    const headers = new Headers(options.headers);
    if (token) {
        headers.set('Authorization', `Bearer ${token}`);
    }

    try {
        const response = await fetch(fullUrl, {
            ...options,
            headers,
            credentials: 'include' // Include cookies for session management
        });

        return response;
    } catch (error) {
        console.error('Fetch error:', error);
        return new Response('Network error', {
            status: 500,
            statusText: 'Network error'
        });
    }
}