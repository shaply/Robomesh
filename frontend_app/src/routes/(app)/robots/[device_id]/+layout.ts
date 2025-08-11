import { browser } from "$app/environment";
import { getRobotById } from "$lib/backend/get_robots.js";
import { error } from '@sveltejs/kit';

export async function load({ params }) {
    const { device_id } = params;

    if (browser) {
        // Validate device_id format (e.g., alphanumeric)
        if (!device_id || !/^[0-9a-zA-Z]+$/.test(device_id)) {
            error(400, 'Invalid device ID format');
        }

        const robot = await getRobotById(device_id);
        if (!robot) {
            error(404, 'Robot not found');
        }

        return {
            robot
        };
    }
    return {};
}