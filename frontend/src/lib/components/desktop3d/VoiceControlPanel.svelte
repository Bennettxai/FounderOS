<!--
	VoiceControlPanel — thin wrapper around shared OsaOrb for the 3D Desktop.
	All behavior lives in OsaOrb.svelte.
-->

<script lang="ts">
	import OsaOrb from '$lib/components/osa/OsaOrb.svelte';

	interface Props {
		isListening: boolean;
		isSpeaking: boolean;
		onToggleListening: () => void;
	}

	let { isListening, isSpeaking, onToggleListening }: Props = $props();
</script>

<OsaOrb {isListening} {isSpeaking} {onToggleListening} />
