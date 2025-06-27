<script lang="ts">

    let title: string = "Card Title";

</script>

<div class="card">
    <h2 class="card-title">{title}</h2>
    <p class="card-content"><slot></slot></p>
</div>

<style>
    .card {
        border: 1px solid #ccc;
        border-radius: 8px;
        padding: 16px;
        margin: 16px 0;
    }

    .card-title {
        font-size: 1.5em;
        margin-bottom: 8px;
    }

    .card-content {
        font-size: 1em;
        color: #555;
    }
</style>