import { PUBLIC_BACKEND_IP, PUBLIC_BACKEND_PORT } from "$env/static/public";
import { API_AUTH_TOKEN } from "$lib/const.js";
import { browser } from "$app/environment";

export async function fetchBackend(url: string, options: RequestInit = {}): Promise<Response> {
    if (!PUBLIC_BACKEND_IP || !PUBLIC_BACKEND_PORT) {
        throw new Error('Backend IP or port is not defined in environment variables');
    }

    const fullUrl = `http://${PUBLIC_BACKEND_IP}:${PUBLIC_BACKEND_PORT}${url}`;

    const headers = new Headers(options.headers);
    if (browser) {
        const token = localStorage.getItem(`${API_AUTH_TOKEN}`);
        if (token) {
            headers.set('Authorization', `Bearer ${token}`);
        }
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 10000);

    try {
        const response = await fetch(fullUrl, {
            ...options,
            headers,
            credentials: 'include',
            signal: controller.signal
        });

        // Redirect to login on auth failure (skip for login/logout endpoints)
        if ((response.status === 401 || response.status === 403) && browser) {
            const isAuthEndpoint = url.startsWith('/auth/login') || url.startsWith('/auth/logout');
            if (!isAuthEndpoint) {
                localStorage.removeItem(`${API_AUTH_TOKEN}`);
                window.location.href = '/login';
            }
        }

        return response;
    } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
            return new Response('Request timed out', {
                status: 408,
                statusText: 'Request Timeout'
            });
        }
        console.error('Fetch error:', error);
        return new Response('Network error', {
            status: 500,
            statusText: 'Network error'
        });
    } finally {
        clearTimeout(timeout);
    }
}