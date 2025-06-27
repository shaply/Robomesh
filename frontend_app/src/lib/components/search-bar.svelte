<script lang="ts">
    export let placeholder: string = "Search bar...";
    export let defaultValue: string = "";
    export let onChange: (value: string) => void = () => {};
</script>

<input
    class="search-bar"
    placeholder="{placeholder}"
    bind:value="{defaultValue}"
    on:input="{(e) => onChange((e.target as HTMLInputElement)?.value ?? '')}"
    {...$$restProps}
>

<style>
    .search-bar {
        border: 1px solid black;
        border-radius: 10px;
        padding: 0.5rem 1rem;
        margin: 0.5rem;
        background-color: #93A3A9;
        color: white;
        font-size: 1.2rem;
        font-weight: 500;
        width: 100%;
    }
    .search-bar::placeholder {
        color: #DADADA;
    }
    .search-bar:focus {
        outline: none;
        border-color: #5C7A8A;
        box-shadow: 0 0 0 2px rgba(92, 122, 138, 0.3);
    }
</style>