<script>
    import { onMount } from "svelte";
    import { goto } from "$app/navigation";

    onMount(() => {
        // Redirect to the login page after logout
        goto("/login");
    });
</script>
