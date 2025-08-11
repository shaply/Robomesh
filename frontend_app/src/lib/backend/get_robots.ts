import type { BaseRobot } from "$lib/types.js";
import { fetchBackend } from "./fetch.js";

export async function getRobots(): Promise<BaseRobot[] | null> {
    try {
        const response = await fetchBackend('/robot', {
            method: 'GET'
        });
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
        const response = await fetchBackend(`/robot/robot/${device_id}`, {
            method: 'GET'
        });
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