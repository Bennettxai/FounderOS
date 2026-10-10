/**
 * View Type Definitions for App Templates
 * These define the different ways data can be visualized.
 */

import type { Field } from './field';

/** Supported view types */
export type ViewType =
  | 'table'
  | 'card'
  | 'kanban'
  | 'calendar'
  | 'timeline'
  | 'gallery'
  | 'chart';

/** Sort direction */
export type SortDirection = 'asc' | 'desc';

/** Sort configuration */
export interface SortConfig {
  fieldId: string;
  direction: SortDirection;
}

/** Filter operator types */
export type FilterOperator =
  | 'equals'
  | 'not_equals'
  | 'contains'
  | 'not_contains'
  | 'starts_with'
  | 'ends_with'
  | 'is_empty'
  | 'is_not_empty'
  | 'greater_than'
  | 'less_than'
  | 'greater_or_equal'
  | 'less_or_equal'
  | 'between'
  | 'in'
  | 'not_in';

/** Single filter condition */
export interface FilterCondition {
  id: string;
  fieldId: string;
  operator: FilterOperator;
  value: unknown;
  value2?: unknown; // For 'between' operator
}

/** Filter group (AND/OR logic) */
export interface FilterGroup {
  id: string;
  operator: 'and' | 'or';
  conditions: (FilterCondition | FilterGroup)[];
}

/** Base view configuration */
export interface BaseViewConfig {
  id: string;
  name: string;
  type: ViewType;
  description?: string;
  isDefault?: boolean;
  filters?: FilterGroup;
  sort?: SortConfig[];
  hiddenFields?: string[];
}

/** Table view configuration */
export interface TableViewConfig extends BaseViewConfig {
  type: 'table';
  density?: 'compact' | 'comfortable' | 'spacious';
  showRowNumbers?: boolean;
  showCheckboxes?: boolean;
  frozenColumns?: number;
  columnWidths?: Record<string, number>;
  columnOrder?: string[];
  groupBy?: string;
  showGroupCounts?: boolean;
  enableInlineEdit?: boolean;
  stripedRows?: boolean;
}

/** Card view configuration */
export interface CardViewConfig extends BaseViewConfig {
  type: 'card';
  titleField: string;
  subtitleField?: string;
  imageField?: string;
  badgeField?: string;
  columns?: number;
  cardSize?: 'small' | 'medium' | 'large';
  showDescription?: boolean;
  descriptionField?: string;
}

/** Kanban view configuration */
export interface KanbanViewConfig extends BaseViewConfig {
  type: 'kanban';
  groupByField: string;
  titleField: string;
  subtitleField?: string;
  columnColors?: Record<string, string>;
  showColumnCounts?: boolean;
  collapsedColumns?: string[];
  cardFields?: string[];
  allowDragDrop?: boolean;
  wipLimits?: Record<string, number>;
}

/** Calendar view configuration */
export interface CalendarViewConfig extends BaseViewConfig {
  type: 'calendar';
  startDateField: string;
  endDateField?: string;
  titleField: string;
  colorField?: string;
  defaultView?: 'month' | 'week' | 'day';
  weekStartsOn?: 0 | 1 | 2 | 3 | 4 | 5 | 6;
}

/** Timeline view configuration */
export interface TimelineViewConfig extends BaseViewConfig {
  type: 'timeline';
  startDateField: string;
  endDateField: string;
  titleField: string;
  groupByField?: string;
  colorField?: string;
  showMilestones?: boolean;
}

/** Gallery view configuration */
export interface GalleryViewConfig extends BaseViewConfig {
  type: 'gallery';
  imageField: string;
  titleField: string;
  subtitleField?: string;
  aspectRatio?: 'square' | '4:3' | '16:9' | 'auto';
  columns?: number;
  gap?: number;
}

/** Chart type */
export type ChartType = 'bar' | 'line' | 'pie' | 'donut' | 'area' | 'scatter';

/** Chart view configuration */
export interface ChartViewConfig extends BaseViewConfig {
  type: 'chart';
  chartType: ChartType;
  xAxisField: string;
  yAxisField: string;
  groupByField?: string;
  colorField?: string;
  aggregation?: 'sum' | 'count' | 'average' | 'min' | 'max';
  showLegend?: boolean;
  showGrid?: boolean;
  showLabels?: boolean;
}

/** Union type of all view configurations */
export type ViewConfig =
  | TableViewConfig
  | CardViewConfig
  | KanbanViewConfig
  | CalendarViewConfig
  | TimelineViewConfig
  | GalleryViewConfig
  | ChartViewConfig;

/** Pagination configuration */
export interface PaginationConfig {
  page: number;
  pageSize: number;
  total: number;
  pageSizeOptions?: number[];
}

/** Selection state */
export interface SelectionState {
  selectedIds: Set<string>;
  isAllSelected: boolean;
  isIndeterminate: boolean;
}

/** Column resize event */
export interface ColumnResizeEvent {
  fieldId: string;
  width: number;
}

/** Row click event */
export interface RowClickEvent {
  recordId: string;
  record: Record<string, unknown>;
  fieldId?: string;
}

/** Bulk action */
export interface BulkAction {
  id: string;
  label: string;
  icon?: string;
  variant?: 'default' | 'danger';
  action: (selectedIds: string[]) => void | Promise<void>;
}

/** Quick filter */
export interface QuickFilter {
  id: string;
  label: string;
  icon?: string;
  filter: FilterGroup;
  count?: number;
}

/** Saved view (user-created) */
export interface SavedView {
  id: string;
  name: string;
  config: ViewConfig;
  isPersonal: boolean;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}
