import { getRobots } from '$lib/backend/get_robots.js';
import { json } from '@sveltejs/kit';

export const GET = async () => {
    try {
        const robots = await getRobots();
        return json(robots || []);
    } catch (error) {
        console.error('Error fetching robots:', error);
        return json([], { status: 500 });
    }
};
