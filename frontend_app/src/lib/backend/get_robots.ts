import type { ActiveRobot, BaseRobot, RegisteredRobot, PendingRobot } from "$lib/types.js";
import { fetchBackend } from "./fetch.js";

// Maps an ActiveRobot (from Redis) into the BaseRobot shape used by the UI.
function activeToBase(r: ActiveRobot): BaseRobot {
    return {
        device_id: r.uuid,
        name: r.uuid,
        ip: r.ip,
        robot_type: r.device_type,
        status: "online",
        last_seen: r.connected_at,
    };
}

export async function getRobots(): Promise<BaseRobot[] | null> {
    try {
        const response = await fetchBackend('/robot', { method: 'GET' });
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const active = (await response.json()) as ActiveRobot[];
        if (!active) return [];
        return active.map(activeToBase);
    } catch (error) {
        console.error('Error fetching robots:', error);
        return null;
    }
}

export async function getRobotById(device_id: string): Promise<BaseRobot | null> {
    try {
        const response = await fetchBackend(`/robot/${device_id}`, { method: 'GET' });
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        if (!data.online) return null;
        return {
            device_id: data.uuid,
            name: data.uuid,
            ip: data.ip,
            robot_type: data.device_type,
            status: "online",
            last_seen: data.connected_at,
        };
    } catch (error) {
        console.error('Error fetching robot by ID:', error);
        return null;
    }
}

// Fetch all robots from PostgreSQL registry (permanent).
export async function getRegisteredRobots(): Promise<RegisteredRobot[] | null> {
    try {
        const response = await fetchBackend('/provision', { method: 'GET' });
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const robots = (await response.json()) as RegisteredRobot[];
        return robots ?? [];
    } catch (error) {
        console.error('Error fetching registered robots:', error);
        return null;
    }
}

// Fetch all pending registration requests from Redis.
export async function getPendingRobots(): Promise<PendingRobot[] | null> {
    try {
        const response = await fetchBackend('/register/pending', { method: 'GET' });
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const robots = (await response.json()) as PendingRobot[];
        return robots ?? [];
    } catch (error) {
        console.error('Error fetching pending robots:', error);
        return null;
    }
}

// Provision a new robot (add public key to PostgreSQL).
export async function provisionRobot(uuid: string, publicKey: string, deviceType: string): Promise<boolean> {
    try {
        const response = await fetchBackend('/provision', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ uuid, public_key: publicKey, device_type: deviceType }),
        });
        return response.ok;
    } catch (error) {
        console.error('Error provisioning robot:', error);
        return false;
    }
}

// Blacklist/unblacklist a robot.
export async function blacklistRobot(uuid: string, blacklisted: boolean): Promise<boolean> {
    try {
        const response = await fetchBackend(`/provision/${uuid}/blacklist`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ blacklisted }),
        });
        return response.ok;
    } catch (error) {
        console.error('Error updating blacklist:', error);
        return false;
    }
}
