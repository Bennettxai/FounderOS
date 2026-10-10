<script lang="ts">
	import { desktopSettings } from '$lib/stores/desktopStore';
	import IconBrandingSection from './IconBrandingSection.svelte';
	import IconSizeSection from './IconSizeSection.svelte';
	import IconStyleSection from './IconStyleSection.svelte';

	let companyNameInput = $state($desktopSettings.companyName);

	function handleSaveName(name: string) {
		desktopSettings.setCompanyName(name);
	}
</script>

<IconBrandingSection bind:companyNameInput onSave={handleSaveName} />
<IconSizeSection />
<IconStyleSection />
