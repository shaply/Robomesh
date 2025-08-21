import type { SvelteComponent } from 'svelte';

// Import all robot type components
import DefaultRobotComponent from './components/DefaultRobotComponent.svelte';
// TODO: Import your robot components here
// import DroneComponent from './components/DroneComponent.svelte';
// import VacuumComponent from './components/VacuumComponent.svelte';
// import ArmComponent from './components/ArmComponent.svelte';

export type RobotComponent = any; // Using any for Svelte 5 compatibility

export interface RobotTypeConfig {
  component: RobotComponent;
  displayName: string;
  description?: string;
  icon?: string; // Icon class or emoji
  capabilities?: {
    realTimeControl?: boolean;
    diagnostics?: boolean;
    configuration?: boolean;
    logs?: boolean;
    camera?: boolean;
  };
  // Default actions this robot type supports
  defaultActions?: string[];
}

// ========================================
// 🤖 ROBOT TYPE REGISTRY
// ========================================
// Add new robot types here when you create them!

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
  
  // TODO: Add your robot types here!
  // 'drone': {
  //   component: DroneComponent,
  //   displayName: 'Aerial Drone',
  //   description: 'Flying robot with camera and GPS',
  //   icon: '🚁',
  //   capabilities: {
  //     realTimeControl: true,
  //     camera: true,
  //     diagnostics: true
  //   },
  //   defaultActions: ['takeoff', 'land', 'hover', 'return-home']
  // },
  
  // 'vacuum': {
  //   component: VacuumComponent,
  //   displayName: 'Vacuum Robot',
  //   description: 'Autonomous cleaning robot',
  //   icon: '🧹',
  //   capabilities: {
  //     realTimeControl: true,
  //     diagnostics: true,
  //     configuration: true
  //   },
  //   defaultActions: ['start-cleaning', 'pause', 'dock', 'spot-clean']
  // },
  
  // 'arm': {
  //   component: ArmComponent,
  //   displayName: 'Robotic Arm',
  //   description: 'Multi-axis manipulator arm',
  //   icon: '🦾',
  //   capabilities: {
  //     realTimeControl: true,
  //     diagnostics: true,
  //     configuration: true
  //   },
  //   defaultActions: ['home', 'calibrate', 'emergency-stop']
  // }
};

// ========================================
// 🛠️ HELPER FUNCTIONS
// ========================================

/**
 * Get robot component configuration by type
 */
export function getRobotComponent(robotType: string): RobotTypeConfig {
  const config = robotTypeRegistry[robotType];
  if (!config) {
    console.warn(`Unknown robot type: ${robotType}, using default`);
    return robotTypeRegistry['default'];
  }
  return config;
}

/**
 * Get all registered robot types
 */
export function getAvailableRobotTypes(): string[] {
  return Object.keys(robotTypeRegistry);
}

/**
 * Get robot types that support a specific capability
 */
export function getRobotTypesByCapability(capability: keyof RobotTypeConfig['capabilities']): string[] {
  return Object.entries(robotTypeRegistry)
    .filter(([_, config]) => config.capabilities?.[capability] === true)
    .map(([type]) => type);
}

/**
 * Check if a robot type supports a specific capability
 */
export function robotSupportsCapability(
  robotType: string, 
  capability: keyof RobotTypeConfig['capabilities']
): boolean {
  const config = getRobotComponent(robotType);
  return config.capabilities?.[capability] === true;
}
