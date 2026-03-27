<script lang="ts">
    export let placeholder: string = "Search robots...";
    export let defaultValue: string = "";
    export let onChange: (value: string) => void = () => {};
</script>

<div class="search-wrapper">
    <svg class="search-icon" viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
        <path fill-rule="evenodd" d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z" clip-rule="evenodd" />
    </svg>
    <input
        class="search-input"
        placeholder="{placeholder}"
        bind:value="{defaultValue}"
        on:input="{(e) => onChange((e.target as HTMLInputElement)?.value ?? '')}"
        {...$$restProps}
    >
</div>

<style>
    .search-wrapper {
        position: relative;
        width: 100%;
    }

    .search-icon {
        position: absolute;
        left: 0.75rem;
        top: 50%;
        transform: translateY(-50%);
        color: var(--text-muted);
        pointer-events: none;
    }

    .search-input {
        width: 100%;
        padding: 0.5rem 0.75rem 0.5rem 2.25rem;
        background: var(--bg-elevated);
        border: 1px solid var(--border);
        border-radius: var(--radius-sm);
        color: var(--text-primary);
        font-size: 0.9rem;
        outline: none;
        transition: border-color 0.15s, box-shadow 0.15s;
    }

    .search-input::placeholder {
        color: var(--text-muted);
    }

    .search-input:focus {
        border-color: var(--accent);
        box-shadow: 0 0 0 3px var(--accent-muted);
    }
</style>
