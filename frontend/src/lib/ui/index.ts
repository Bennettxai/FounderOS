// UI Primitives - Svelte/Bits UI Components
// Based on AFFiNE patterns, adapted for BusinessOS
// Synced from BusinessOS2 with additional pattern components

// Core Components
export { default as Button } from "./button/Button.svelte";
export { default as Input } from "./input/Input.svelte";
export { default as Loading } from "./loading/Loading.svelte";
export { default as Skeleton } from "./skeleton/Skeleton.svelte";
export { default as Separator } from "./separator/Separator.svelte";

// Overlay Components
export { default as Modal } from "./modal/Modal.svelte";
export { default as Tooltip } from "./tooltip/Tooltip.svelte";
export { default as Popover } from "./popover/Popover.svelte";

// Menu Components
export { Menu, MenuItem, MenuSeparator, MenuLabel, MenuGroup } from "./menu";

// Tab Components
export { Tabs, TabsList, TabsTrigger, TabsContent } from "./tabs";

// Layout Components
export { default as ScrollArea } from "./scroll-area/ScrollArea.svelte";

// Pattern Components (extracted from BusinessOS2)
export { default as Badge } from "./badge/Badge.svelte";
export { default as Avatar } from "./avatar/Avatar.svelte";
export { default as IconButton } from "./icon-button/IconButton.svelte";
export { default as Card } from "./card/Card.svelte";
export { default as SidebarItem } from "./sidebar-item/SidebarItem.svelte";
export { default as Switch } from "./switch/Switch.svelte";
