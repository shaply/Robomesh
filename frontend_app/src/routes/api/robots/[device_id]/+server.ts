import { getRobotById } from '$lib/backend/get_robots.js';
import { json } from '@sveltejs/kit';

export const GET = async ({ params }) => {
    const { device_id } = params;

    try {
        const robot = await getRobotById(device_id);
        return json(robot || {});
    } catch (error) {
        console.error('Error fetching robot:', error);
        return json({}, { status: 500 });
    }
};
