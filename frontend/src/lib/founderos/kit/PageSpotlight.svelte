<!-- The page-wide glow (FounderOS v1 PageSpotlight): one fixed layer over the
     whole viewport following the cursor via --bn-px/--bn-py on :root (set by
     lens.ts). Mount once in the /os layout. -->
<script lang="ts">
	import './kit.css';
</script>

<span aria-hidden="true" class="bn-spotlight is-page"></span>
