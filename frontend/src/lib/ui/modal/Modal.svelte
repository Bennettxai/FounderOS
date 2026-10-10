<script lang="ts">
	import { Dialog as DialogPrimitive } from 'bits-ui';
	import { type Snippet } from 'svelte';
	import { cn } from '$lib/utils';
	import { X } from 'lucide-svelte';

	type ModalSize = 'sm' | 'default' | 'lg' | 'xl' | 'full';

	interface Props {
		open?: boolean;
		onOpenChange?: (open: boolean) => void;
		title?: string;
		description?: string;
		size?: ModalSize;
		showClose?: boolean;
		closeOnOutsideClick?: boolean;
		closeOnEscape?: boolean;
		class?: string;
		children: Snippet;
		footer?: Snippet;
	}

	let {
		open = $bindable(false),
		onOpenChange,
		title,
		description,
		size = 'default',
		showClose = true,
		closeOnOutsideClick = true,
		closeOnEscape = true,
		class: className = '',
		children,
		footer
	}: Props = $props();

	const sizeStyles: Record<ModalSize, string> = {
		sm: 'max-w-sm',
		default: 'max-w-lg',
		lg: 'max-w-2xl',
		xl: 'max-w-4xl',
		full: 'max-w-[calc(100vw-2rem)] h-[calc(100vh-2rem)]'
	};

	function handleOpenChange(value: boolean) {
		open = value;
		onOpenChange?.(value);
	}
</script>

<DialogPrimitive.Root bind:open onOpenChange={handleOpenChange}>
	<DialogPrimitive.Portal>
		<DialogPrimitive.Overlay
			class={cn(
				'fixed inset-0 z-50 bg-black/80',
				'data-[state=open]:animate-in data-[state=closed]:animate-out',
				'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0'
			)}
		/>
		<DialogPrimitive.Content
			class={cn(
				'fixed left-[50%] top-[50%] z-50 w-full translate-x-[-50%] translate-y-[-50%]',
				'grid gap-4 border bg-background p-6 shadow-lg duration-200',
				'data-[state=open]:animate-in data-[state=closed]:animate-out',
				'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
				'data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
				'data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%]',
				'data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%]',
				'sm:rounded-lg',
				sizeStyles[size],
				className
			)}
			onInteractOutside={(e) => {
				if (!closeOnOutsideClick) e.preventDefault();
			}}
			onEscapeKeydown={(e) => {
				if (!closeOnEscape) e.preventDefault();
			}}
		>
			{#if title || description}
				<div class="flex flex-col space-y-1.5 text-center sm:text-left">
					{#if title}
						<DialogPrimitive.Title class="text-lg font-semibold leading-none tracking-tight">
							{title}
						</DialogPrimitive.Title>
					{/if}
					{#if description}
						<DialogPrimitive.Description class="text-sm text-muted-foreground">
							{description}
						</DialogPrimitive.Description>
					{/if}
				</div>
			{/if}

			<div class="flex-1">
				{@render children()}
			</div>

			{#if footer}
				<div class="flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2">
					{@render footer()}
				</div>
			{/if}

			{#if showClose}
				<DialogPrimitive.Close
					class={cn(
						'absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background',
						'transition-opacity hover:opacity-100 focus:outline-none focus:ring-2',
						'focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none',
						'data-[state=open]:bg-accent data-[state=open]:text-muted-foreground'
					)}
				>
					<X class="h-4 w-4" />
					<span class="sr-only">Close</span>
				</DialogPrimitive.Close>
			{/if}
		</DialogPrimitive.Content>
	</DialogPrimitive.Portal>
</DialogPrimitive.Root>
