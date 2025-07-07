import { BACKEND_IP, BACKEND_PORT } from "$env/static/private";
import { PATH_GET_ROBOTS } from "$lib/backend_paths.js";
import type { BaseRobot } from "$lib/types.js";

export async function getRobots(): Promise<BaseRobot[] | null> {
    try {
        const response = await fetch(`http://${BACKEND_IP}:${BACKEND_PORT}/${PATH_GET_ROBOTS}`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const robots = (await response.json()) as BaseRobot[];
        return robots;
    } catch (error) {
        console.error('Error fetching robots:', error);
        return null;
    }
}

export async function getRobotById(device_id: string): Promise<BaseRobot | null> {
    try {
        const response = await fetch(`http://${BACKEND_IP}:${BACKEND_PORT}/${PATH_GET_ROBOTS}/${device_id}`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const robot = (await response.json()) as BaseRobot;
        return robot;
    } catch (error) {
        console.error('Error fetching robot by ID:', error);
        return null;
    }
}