<script lang="ts">
    export let variant: 'primary' | 'secondary' | 'danger' = 'primary';
    export let disabled: boolean = false;
    export let type: 'button' | 'submit' | 'reset' = 'button';
</script>

<button 
    class="btn btn-{variant}" 
    {disabled} 
    {type}
    on:click
    {...$$restProps}
>
    <slot></slot>
</button>

<style>
    .btn {
        font-size: 1.2rem;
        border: 1px solid transparent;
        padding: 0.5rem 1rem;
        border-radius: 10px;
        cursor: pointer;
        margin: 0.5rem;
        transition: background-color 0.2s ease;
        font-weight: 500;
    }

    .btn:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .btn-primary {
        color: white;
        background-color: #98B9C8;
        border-color: black;
    }

    .btn-primary:hover:not(:disabled) {
        background-color: #7A9BAE;
    }

    .btn-primary:active:not(:disabled) {
        background-color: #5C7A8A;
    }

    .btn-secondary {
        color: #374151;
        background-color: #f3f4f6;
        border-color: #d1d5db;
    }

    .btn-secondary:hover:not(:disabled) {
        background-color: #e5e7eb;
    }

    .btn-danger {
        color: white;
        background-color: #ef4444;
        border-color: #dc2626;
    }

    .btn-danger:hover:not(:disabled) {
        background-color: #dc2626;
    }
</style>