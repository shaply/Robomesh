import { API_AUTH_TOKEN } from "$lib/const.js";
import { redirect } from "@sveltejs/kit";
import { browser } from "$app/environment";
import { fetchBackend } from "$lib/backend/fetch.js";
import { authFailureRedirect } from "$lib/utils/failure_redirect.js";

export async function load() {
    if (browser) {
        await fetchBackend('/auth/logout', {
            method: 'POST'
        })

        authFailureRedirect();
    }
    return {};
}