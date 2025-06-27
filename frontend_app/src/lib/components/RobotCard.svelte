<script lang="ts">
    import type { BaseRobot } from '$lib/types.js';

    export let robot: BaseRobot;
    export let showTitle: boolean = true;

    // Helper function to get status color
    function getStatusColor(status: string): string {
        switch (status.toLowerCase()) {
            case 'online':
            case 'active':
                return '#22c55e'; // green
            case 'offline':
            case 'inactive':
                return '#ef4444'; // red
            case 'maintenance':
            case 'warning':
                return '#f59e0b'; // amber
            default:
                return '#6b7280'; // gray
        }
    }
</script>

<div class="robot-card">
    {#if showTitle}
        <h2 class="robot-name">{robot.name}</h2>
    {/if}
    
    <div class="robot-info">
        <ul class="robot-attributes">
            <li><strong>ID:</strong> {robot.id}</li>
            <li><strong>IP Address:</strong> {robot.ip}</li>
            <li><strong>Robot Type:</strong> {robot.robotType}</li>
            <li>
                <strong>Status:</strong> 
                <span class="status-badge" style="background-color: {getStatusColor(robot.status)}">
                    {robot.status}
                </span>
            </li>
            <li><strong>Device ID:</strong> {robot.deviceId}</li>
        </ul>
    </div>

    <!-- Optional slot for additional content -->
    <div class="robot-actions">
        <slot name="actions" />
    </div>
</div>

<style>
    .robot-card {
        border: 1px solid #e5e7eb;
        border-radius: 12px;
        padding: 20px;
        margin: 16px 0;
        background: white;
        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
        transition: box-shadow 0.2s ease;
    }

    .robot-card:hover {
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    }

    .robot-name {
        font-size: 1.5rem;
        font-weight: 600;
        margin-bottom: 16px;
        color: #1f2937;
        border-bottom: 2px solid #f3f4f6;
        padding-bottom: 8px;
    }

    .robot-info {
        margin-bottom: 16px;
    }

    .robot-attributes {
        list-style: none;
        padding: 0;
        margin: 0;
    }

    .robot-attributes li {
        padding: 8px 0;
        display: flex;
        align-items: center;
        border-bottom: 1px solid #f9fafb;
    }

    .robot-attributes li:last-child {
        border-bottom: none;
    }

    .robot-attributes strong {
        color: #374151;
        min-width: 100px;
        margin-right: 8px;
    }

    .status-badge {
        display: inline-block;
        color: white;
        padding: 4px 8px;
        border-radius: 12px;
        font-size: 0.875rem;
        font-weight: 500;
        text-transform: capitalize;
    }

    .robot-actions {
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid #f3f4f6;
    }

    .robot-actions:empty {
        display: none;
    }
</style>
