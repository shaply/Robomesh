import { browser } from "$app/environment";
import { API_AUTH_TOKEN } from "$lib/const.js";
import { redirect } from "@sveltejs/kit";

export function authFailureRedirect() {
  if (browser) {
    // Redirect to the login page if authentication fails
    console.log("Authentication failed, redirecting to login page.");
    localStorage.removeItem(`${API_AUTH_TOKEN}`);
    throw redirect(302, '/login');
  }
}