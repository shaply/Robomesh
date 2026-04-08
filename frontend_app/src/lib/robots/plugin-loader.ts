import { backendBaseUrl } from "$lib/backend/fetch.js";

const pluginCache = new Map<string, { module: any; loadedAt: number }>();
const PLUGIN_CACHE_TTL = 5 * 60 * 1000; // 5 minutes

/**
 * Returns the base URL for plugin assets.
 */
function pluginBaseUrl(robotType: string): string {
    return `${backendBaseUrl()}/plugins/${robotType}`;
}

/**
 * Dynamically loads a plugin component (robot_card or robot_handler) for a given robot type.
 * Returns null if the plugin is not available (falls back to default).
 */
export async function loadPluginComponent(
    robotType: string,
    componentName: 'robot_card' | 'robot_handler'
): Promise<any | null> {
    const cacheKey = `${robotType}:${componentName}`;

    const cached = pluginCache.get(cacheKey);
    if (cached && (Date.now() - cached.loadedAt) < PLUGIN_CACHE_TTL) {
        return cached.module;
    }

    const url = `${pluginBaseUrl(robotType)}/${componentName}.js`;

    try {
        const module = await import(/* @vite-ignore */ url);
        const component = module.default;
        if (component) {
            pluginCache.set(cacheKey, { module: component, loadedAt: Date.now() });
            return component;
        }
    } catch (e) {
        // Plugin not available for this type — that's fine
        if (import.meta.env.DEV) console.debug(`No ${componentName} plugin for robot type "${robotType}"`);
    }

    return null;
}

/**
 * Checks which robot types have plugins available on the backend.
 */
export async function getAvailablePlugins(): Promise<string[]> {
    try {
        const res = await fetch(
            `${backendBaseUrl()}/plugins/`
        );
        if (!res.ok) return [];
        return await res.json();
    } catch {
        return [];
    }
}

/**
 * Clears the plugin cache (useful during development).
 */
export function clearPluginCache(): void {
    pluginCache.clear();
}
