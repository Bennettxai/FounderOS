/**
 * App Configuration Types for App Templates
 * The AppConfig is the JSON schema that drives the entire app rendering.
 */

import type { Field, RecordData } from './field';
import type { ViewConfig, BulkAction, QuickFilter } from './view';

/** App theme configuration */
export interface AppTheme {
  /** Primary accent color (hex) */
  primaryColor?: string;
  /** Secondary accent color (hex) */
  secondaryColor?: string;
  /** Whether to use dark mode */
  darkMode?: boolean;
  /** Custom CSS variables */
  customTokens?: Record<string, string>;
}

/** App branding */
export interface AppBranding {
  /** App name */
  name: string;
  /** App description */
  description?: string;
  /** App icon (emoji or URL) */
  icon?: string;
  /** App logo URL */
  logo?: string;
  /** Favicon URL */
  favicon?: string;
}

/** Navigation item */
export interface NavItem {
  id: string;
  label: string;
  icon?: string;
  href?: string;
  children?: NavItem[];
  badge?: string | number;
}

/** App navigation configuration */
export interface AppNavigation {
  /** Show sidebar navigation */
  showSidebar?: boolean;
  /** Navigation items */
  items?: NavItem[];
  /** Show search in navigation */
  showSearch?: boolean;
  /** Show user menu */
  showUserMenu?: boolean;
}

/** Toolbar configuration */
export interface ToolbarConfig {
  /** Show search bar */
  showSearch?: boolean;
  /** Search placeholder */
  searchPlaceholder?: string;
  /** Show view switcher */
  showViewSwitcher?: boolean;
  /** Show filter button */
  showFilter?: boolean;
  /** Show sort button */
  showSort?: boolean;
  /** Show export button */
  showExport?: boolean;
  /** Custom actions */
  actions?: {
    id: string;
    label: string;
    icon?: string;
    variant?: 'primary' | 'secondary' | 'ghost';
    action: string; // Action identifier
  }[];
}

/** Dashboard widget configuration */
export interface DashboardWidget {
  id: string;
  type: 'stat' | 'chart' | 'list' | 'activity';
  title: string;
  /** Grid position */
  position: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  config: Record<string, unknown>;
}

/** Dashboard configuration */
export interface DashboardConfig {
  enabled: boolean;
  widgets: DashboardWidget[];
  refreshInterval?: number; // in seconds
}

/** Form configuration */
export interface FormConfig {
  /** Form layout */
  layout?: 'single' | 'double' | 'auto';
  /** Sections */
  sections?: {
    id: string;
    title?: string;
    description?: string;
    fieldIds: string[];
    collapsible?: boolean;
    defaultCollapsed?: boolean;
  }[];
  /** Submit button text */
  submitText?: string;
  /** Cancel button text */
  cancelText?: string;
  /** Show validation inline */
  inlineValidation?: boolean;
}

/** Detail panel configuration */
export interface DetailPanelConfig {
  /** Panel width */
  width?: 'narrow' | 'medium' | 'wide' | 'full';
  /** Show tabs */
  tabs?: {
    id: string;
    label: string;
    icon?: string;
    content: 'fields' | 'activity' | 'comments' | 'related' | 'custom';
  }[];
  /** Header fields (shown prominently) */
  headerFields?: string[];
  /** Show activity feed */
  showActivity?: boolean;
  /** Show comments */
  showComments?: boolean;
}

/** Permission configuration */
export interface PermissionConfig {
  /** Can create new records */
  canCreate?: boolean;
  /** Can edit records */
  canEdit?: boolean;
  /** Can delete records */
  canDelete?: boolean;
  /** Can export data */
  canExport?: boolean;
  /** Can change views */
  canChangeView?: boolean;
  /** Field-level permissions */
  fieldPermissions?: Record<string, {
    canView?: boolean;
    canEdit?: boolean;
  }>;
}

/** Main App Configuration */
export interface AppConfig {
  /** Unique app identifier */
  id: string;

  /** App version */
  version: string;

  /** Branding configuration */
  branding: AppBranding;

  /** Theme configuration */
  theme?: AppTheme;

  /** Navigation configuration */
  navigation?: AppNavigation;

  /** Field definitions */
  fields: Field[];

  /** View configurations */
  views: ViewConfig[];

  /** Default view ID */
  defaultViewId?: string;

  /** Toolbar configuration */
  toolbar?: ToolbarConfig;

  /** Dashboard configuration */
  dashboard?: DashboardConfig;

  /** Form configuration for create/edit */
  form?: FormConfig;

  /** Detail panel configuration */
  detailPanel?: DetailPanelConfig;

  /** Quick filters */
  quickFilters?: QuickFilter[];

  /** Bulk actions */
  bulkActions?: BulkAction[];

  /** Permissions */
  permissions?: PermissionConfig;

  /** Initial data imported from a configured source */
  initialData?: RecordData[];

  /** API endpoints (if connected to backend) */
  api?: {
    baseUrl?: string;
    endpoints?: {
      list?: string;
      get?: string;
      create?: string;
      update?: string;
      delete?: string;
    };
  };

  /** Feature flags */
  features?: {
    enableSearch?: boolean;
    enableFilter?: boolean;
    enableSort?: boolean;
    enableExport?: boolean;
    enableImport?: boolean;
    enableInlineEdit?: boolean;
    enableBulkActions?: boolean;
    enableKeyboardShortcuts?: boolean;
    enableRealtime?: boolean;
  };
}

/** App state (runtime) */
export interface AppState {
  /** Current view ID */
  currentViewId: string;
  /** Current view config */
  currentView: ViewConfig;
  /** Current filters */
  filters?: Record<string, unknown>;
  /** Current sort */
  sort?: { field: string; direction: 'asc' | 'desc' }[];
  /** Current search query */
  searchQuery?: string;
  /** Selected record IDs */
  selectedIds: string[];
  /** Currently open record ID */
  openRecordId?: string;
  /** Is loading */
  isLoading: boolean;
  /** Error state */
  error?: string;
  /** Pagination */
  pagination: {
    page: number;
    pageSize: number;
    total: number;
  };
}

/** Sample app configs for different use cases */
export type AppTemplate =
  | 'crm'
  | 'project-tracker'
  | 'inventory'
  | 'hr'
  | 'content-calendar'
  | 'bug-tracker'
  | 'custom';

/** Helper to create a basic app config */
export function createAppConfig(
  template: AppTemplate,
  overrides?: Partial<AppConfig>
): AppConfig {
  const baseConfig: AppConfig = {
    id: `app-${Date.now()}`,
    version: '1.0.0',
    branding: {
      name: 'New App',
      icon: '📱',
    },
    fields: [],
    views: [],
    features: {
      enableSearch: true,
      enableFilter: true,
      enableSort: true,
      enableExport: true,
      enableInlineEdit: true,
      enableBulkActions: true,
      enableKeyboardShortcuts: true,
    },
  };

  return { ...baseConfig, ...overrides };
}
