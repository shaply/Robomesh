import { redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { API_AUTH_TOKEN } from '$lib/const.js';
import { authFailureRedirect } from '$lib/utils/failure_redirect.js';

export async function load({ url }) {
    // List of public routes that don't require authentication
    const publicRoutes = ['/login', '/register'];
    
    // Check if current route is public
    const isPublicRoute = publicRoutes.some(route => url.pathname.startsWith(route));
    
    if (isPublicRoute) {
        return {};
    }

    // Only check auth on client-side where localStorage is available
    if (browser) {
        const authToken = localStorage.getItem(`${API_AUTH_TOKEN}`); // or whatever your token key is
        
        if (!authToken) {
            // User is not authenticated, redirect to auth page
            authFailureRedirect();
        }
        
        // TODO: Validate token with your backend here if needed
        // const isValidToken = await validateToken(authToken);
        // if (!isValidToken) {
        //     localStorage.removeItem('auth-token');
        //     redirect(302, '/auth');
        // }
    }
    
    // Return user data if needed
    return {
    };
}
