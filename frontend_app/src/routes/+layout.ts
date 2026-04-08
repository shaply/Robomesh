import { browser } from '$app/environment';
import { API_AUTH_TOKEN } from '$lib/const.js';
import { authFailureRedirect } from '$lib/utils/failure_redirect.js';

export async function load({ url, fetch }) {
    const publicRoutes = ['/login', '/register'];
    const isPublicRoute = publicRoutes.some(route => url.pathname.startsWith(route));

    if (isPublicRoute) {
        return {};
    }

    if (browser) {
        const authToken = localStorage.getItem(`${API_AUTH_TOKEN}`);

        if (!authToken) {
            authFailureRedirect();
            return {};
        }

        // Validate token with backend on initial page load
        try {
            const { backendBaseUrl } = await import('$lib/backend/fetch.js');
            const response = await fetch(`${backendBaseUrl()}/auth`, {
                method: 'GET',
                headers: { 'Authorization': `Bearer ${authToken}` },
            });
            if (!response.ok) {
                localStorage.removeItem(`${API_AUTH_TOKEN}`);
                authFailureRedirect();
            }
        } catch {
            // Network error — allow through, the app layout will handle retry
        }
    }

    return {};
}
