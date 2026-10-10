/**
 * Field Type Definitions for App Templates
 * These define the types of data fields that can be used in generated apps.
 */

/** All supported field types */
export type FieldType =
  | 'text'
  | 'number'
  | 'currency'
  | 'date'
  | 'datetime'
  | 'checkbox'
  | 'select'
  | 'multiselect'
  | 'status'
  | 'email'
  | 'phone'
  | 'url'
  | 'rating'
  | 'progress'
  | 'user'
  | 'relation'
  | 'formula'
  | 'richtext'
  | 'file'
  | 'image';

/** Base field definition */
export interface BaseField {
  id: string;
  name: string;
  label?: string;
  type: FieldType;
  description?: string;
  required?: boolean;
  readonly?: boolean;
  hidden?: boolean;
  width?: number; // Column width in pixels
  minWidth?: number;
  maxWidth?: number;
  config?: LegacyFieldConfig;
}

/** Text field */
export interface TextField extends BaseField {
  type: 'text';
  placeholder?: string;
  maxLength?: number;
  multiline?: boolean;
}

/** Number field */
export interface NumberField extends BaseField {
  type: 'number';
  min?: number;
  max?: number;
  step?: number;
  precision?: number;
  format?: 'decimal' | 'integer' | 'percent';
  prefix?: string;
  suffix?: string;
}

/** Currency field */
export interface CurrencyField extends BaseField {
  type: 'currency';
  currency?: 'USD' | 'EUR' | 'GBP' | 'JPY' | 'CNY' | string;
  locale?: string;
  precision?: number;
}

/** Date field */
export interface DateField extends BaseField {
  type: 'date' | 'datetime';
  format?: string; // e.g., 'MM/DD/YYYY', 'YYYY-MM-DD'
  min?: string;
  max?: string;
  includeTime?: boolean;
}

/** Checkbox field */
export interface CheckboxField extends BaseField {
  type: 'checkbox';
  label?: string;
}

/** Select/Status field option */
export interface SelectOption {
  value: string;
  label: string;
  color?: string;
  icon?: string;
}

export type StatusOption = SelectOption;

export interface LegacyFieldConfig {
  min?: number;
  max?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string | RegExp;
  symbol?: string;
  options?: SelectOption[];
}

/** Select field */
export interface SelectField extends BaseField {
  type: 'select' | 'multiselect';
  options: SelectOption[];
  allowCustom?: boolean;
}

/** Status field (special select with visual indicators) */
export interface StatusField extends BaseField {
  type: 'status';
  options: SelectOption[];
}

/** Email field */
export interface EmailField extends BaseField {
  type: 'email';
  placeholder?: string;
}

/** Phone field */
export interface PhoneField extends BaseField {
  type: 'phone';
  format?: string;
  placeholder?: string;
}

/** URL field */
export interface URLField extends BaseField {
  type: 'url';
  placeholder?: string;
  showFavicon?: boolean;
}

/** Rating field */
export interface RatingField extends BaseField {
  type: 'rating';
  max?: number; // Default 5
  allowHalf?: boolean;
  icon?: 'star' | 'heart' | 'thumb';
}

/** Progress field */
export interface ProgressField extends BaseField {
  type: 'progress';
  min?: number;
  max?: number;
  showLabel?: boolean;
  color?: string;
}

/** User field */
export interface UserField extends BaseField {
  type: 'user';
  multiple?: boolean;
  showAvatar?: boolean;
  showEmail?: boolean;
}

/** Relation field (link to another table) */
export interface RelationField extends BaseField {
  type: 'relation';
  relatedTableId: string;
  relatedTableName: string;
  displayField: string;
  multiple?: boolean;
}

/** Formula field */
export interface FormulaField extends BaseField {
  type: 'formula';
  formula: string;
  resultType: 'text' | 'number' | 'date' | 'boolean';
}

/** Rich text field */
export interface RichTextField extends BaseField {
  type: 'richtext';
  toolbar?: ('bold' | 'italic' | 'underline' | 'link' | 'list' | 'heading')[];
}

/** File field */
export interface FileField extends BaseField {
  type: 'file';
  accept?: string[];
  maxSize?: number; // in bytes
  multiple?: boolean;
}

/** Image field */
export interface ImageField extends BaseField {
  type: 'image';
  accept?: string[];
  maxSize?: number;
  aspectRatio?: string;
  thumbnailSize?: number;
}

/** Union type of all field types */
export type Field =
  | TextField
  | NumberField
  | CurrencyField
  | DateField
  | CheckboxField
  | SelectField
  | StatusField
  | EmailField
  | PhoneField
  | URLField
  | RatingField
  | ProgressField
  | UserField
  | RelationField
  | FormulaField
  | RichTextField
  | FileField
  | ImageField;

/** Record data type (dynamic key-value) */
export type RecordData = Record<string, unknown>;

/** User type for user fields */
export interface User {
  id: string;
  name: string;
  email?: string;
  avatar?: string;
}

/** File attachment type */
export interface FileAttachment {
  id: string;
  name: string;
  url: string;
  size: number;
  type: string;
  thumbnailUrl?: string;
}

/** Validation error */
export interface FieldValidationError {
  fieldId: string;
  message: string;
  type: 'required' | 'format' | 'range' | 'custom';
}

/** Field change event */
export interface FieldChangeEvent {
  fieldId: string;
  oldValue: unknown;
  newValue: unknown;
  recordId?: string;
}
