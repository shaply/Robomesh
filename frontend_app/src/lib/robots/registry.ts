import type { SvelteComponent } from 'svelte';
import { loadPluginComponent } from './plugin-loader.js';

// Built-in default component
import DefaultRobotComponent from './components/DefaultRobotComponent.svelte';

export type RobotComponent = any; // Using any for Svelte 5 compatibility

export interface RobotTypeConfig {
  component: RobotComponent;
  handlerComponent?: RobotComponent;
  displayName: string;
  description?: string;
  icon?: string;
  capabilities?: {
    realTimeControl?: boolean;
    diagnostics?: boolean;
    configuration?: boolean;
    logs?: boolean;
    camera?: boolean;
  };
  defaultActions?: string[];
  isPlugin?: boolean;
}

// ========================================
// ROBOT TYPE REGISTRY
// ========================================
// Built-in types are registered here.
// Plugin types are loaded dynamically at runtime from the backend.

export const robotTypeRegistry: Record<string, RobotTypeConfig> = {
  'default': {
    component: DefaultRobotComponent,
    displayName: 'Generic Robot',
    description: 'A standard robot interface',
    icon: '🤖',
    capabilities: {
      realTimeControl: true,
      diagnostics: true,
      configuration: true,
      logs: true
    },
    defaultActions: ['start', 'stop', 'reset']
  },
};

// ========================================
// HELPER FUNCTIONS
// ========================================

/**
 * Get robot component configuration by type.
 * For built-in types, returns immediately.
 * For dynamic plugin types, use getRobotComponentAsync.
 */
export function getRobotComponent(robotType: string): RobotTypeConfig {
  const config = robotTypeRegistry[robotType];
  if (!config) {
    return robotTypeRegistry['default'];
  }
  return config;
}

/**
 * Async version that attempts to load dynamic plugin components.
 * Falls back to built-in registry, then to default.
 */
export async function getRobotComponentAsync(robotType: string): Promise<RobotTypeConfig> {
  // Check built-in registry first
  if (robotTypeRegistry[robotType]) {
    return robotTypeRegistry[robotType];
  }

  // Try loading from plugin system
  const cardComponent = await loadPluginComponent(robotType, 'robot_card');
  const handlerComponent = await loadPluginComponent(robotType, 'robot_handler');

  if (cardComponent || handlerComponent) {
    const config: RobotTypeConfig = {
      component: cardComponent || DefaultRobotComponent,
      handlerComponent: handlerComponent || undefined,
      displayName: robotType.charAt(0).toUpperCase() + robotType.slice(1).replace(/_/g, ' '),
      description: `Dynamically loaded ${robotType} handler`,
      isPlugin: true,
    };

    // Cache in registry for future sync access
    robotTypeRegistry[robotType] = config;
    return config;
  }

  return robotTypeRegistry['default'];
}

/**
 * Get all registered robot types (built-in only; plugins are loaded on demand).
 */
export function getAvailableRobotTypes(): string[] {
  return Object.keys(robotTypeRegistry);
}

/**
 * Get robot types that support a specific capability.
 */
export function getRobotTypesByCapability(capability: keyof RobotTypeConfig['capabilities']): string[] {
  return Object.entries(robotTypeRegistry)
    .filter(([_, config]) => config.capabilities?.[capability] === true)
    .map(([type]) => type);
}

/**
 * Check if a robot type supports a specific capability.
 */
export function robotSupportsCapability(
  robotType: string,
  capability: keyof RobotTypeConfig['capabilities']
): boolean {
  const config = getRobotComponent(robotType);
  return config.capabilities?.[capability] === true;
}
