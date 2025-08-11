import { browser } from "$app/environment";
import { API_AUTH_TOKEN } from "$lib/const.js";
import { redirect } from "@sveltejs/kit";

export async function load() {
    if (browser) {
        if (localStorage.getItem(`${API_AUTH_TOKEN}`)) {
            redirect(302, '/');
        }
    }
}