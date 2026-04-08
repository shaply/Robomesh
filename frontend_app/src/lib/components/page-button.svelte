<script lang="ts">
    import type { Snippet } from 'svelte';
    import type { HTMLButtonAttributes } from 'svelte/elements';

    interface Props extends HTMLButtonAttributes {
        variant?: 'primary' | 'secondary' | 'danger';
        children?: Snippet;
    }

    let { variant = 'primary', children, ...rest }: Props = $props();
</script>

<button
    class="btn btn-{variant}"
    {...rest}
>
    {@render children?.()}
</button>

<style>
    .btn {
        font-size: 0.85rem;
        border: 1px solid var(--border);
        padding: 0.45rem 1rem;
        border-radius: var(--radius-sm);
        cursor: pointer;
        transition: background-color 0.12s, color 0.12s, border-color 0.12s;
        font-weight: 500;
        white-space: nowrap;
    }

    .btn:disabled {
        opacity: 0.45;
        cursor: not-allowed;
    }

    .btn-primary {
        color: #0b1120;
        background: var(--accent);
        border-color: transparent;
    }

    .btn-primary:hover:not(:disabled) {
        background: var(--accent-hover);
    }

    .btn-secondary {
        color: var(--text-secondary);
        background: var(--bg-elevated);
        border-color: var(--border);
    }

    .btn-secondary:hover:not(:disabled) {
        background: var(--bg-hover);
        color: var(--text-primary);
    }

    .btn-danger {
        color: white;
        background: var(--error);
        border-color: transparent;
    }

    .btn-danger:hover:not(:disabled) {
        background: #dc2626;
    }
</style>
