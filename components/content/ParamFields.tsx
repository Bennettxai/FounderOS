'use client';

import { useState } from 'react';
import { ChevronDown } from 'lucide-react';
import { visibleParams, type HfModel, type HfParam } from '@/lib/higgsfield/catalog';

/**
 * Schema-driven parameter fields: renders a model's real parameter schema —
 * selects for options, steppers/sliders for ranges, toggles for bools —
 * split into a primary block and a collapsed advanced block by usage
 * frequency. Shared by the Composer and the pipeline node editor so every
 * surface exposes the full parameter set with zero per-model UI work.
 */

export type ParamValues = Record<string, string | number | boolean | string[]>;

const inputCls =
  'w-full rounded-[9px] border border-[rgb(var(--mist-rgb) / 0.14)] bg-[var(--surface-2)] px-3 py-2 text-[13px] text-os-text outline-none';

function Field({
  param,
  value,
  onChange,
}: {
  param: HfParam;
  value: string | number | boolean | string[] | undefined;
  onChange: (name: string, value: string | number | boolean | string[]) => void;
}) {
  const spaced = param.name.replaceAll('_', ' ');
  const label = spaced.charAt(0).toUpperCase() + spaced.slice(1);

  if (param.type === 'bool') {
    const on = typeof value === 'boolean' ? value : param.default === true;
    return (
      <label className="flex items-center justify-between gap-3 py-1">
        <span className="text-[12px] text-os-muted" title={param.description}>
          {label}
        </span>
        <button
          type="button"
          role="switch"
          aria-checked={on}
          aria-label={label}
          onClick={() => onChange(param.name, !on)}
          className={`relative h-5 w-9 shrink-0 rounded-full border transition-colors ${
            on ? 'border-os-accent/50 bg-os-accent/25' : 'border-[rgb(var(--mist-rgb) / 0.18)] bg-black/25'
          }`}
        >
          <span
            className={`absolute top-[3px] h-3 w-3 rounded-full transition-all ${
              on ? 'left-[19px] bg-os-accent' : 'left-[3px] bg-os-dim'
            }`}
          />
        </button>
      </label>
    );
  }

  if (param.options && param.options.length > 0) {
    const current = value !== undefined ? String(value) : param.default !== undefined ? String(param.default) : '';
    // Short option sets read better as a chip row; long ones as a select.
    if (param.options.length <= 5) {
      return (
        <div className="py-1">
          <span className="text-[12px] text-os-muted" title={param.description}>
            {label}
          </span>
          <div className="mt-1.5 flex flex-wrap gap-1.5">
            {param.options.map((opt) => {
              const active = String(opt) === current;
              return (
                <button
                  key={String(opt)}
                  type="button"
                  onClick={() =>
                    onChange(param.name, typeof param.options![0] === 'number' ? Number(opt) : String(opt))
                  }
                  className={`rounded-[8px] border px-2.5 py-1 font-mono text-[11.5px] transition-colors ${
                    active
                      ? 'glass-chip gl-on text-os-text'
                      : 'border-[rgb(var(--mist-rgb) / 0.14)] text-os-muted hover:text-os-text'
                  }`}
                >
                  {String(opt)}
                </button>
              );
            })}
          </div>
        </div>
      );
    }
    return (
      <label className="flex flex-col gap-1.5 py-1">
        <span className="text-[12px] text-os-muted" title={param.description}>
          {label}
        </span>
        <select
          value={current}
          onChange={(e) =>
            onChange(param.name, typeof param.options![0] === 'number' ? Number(e.target.value) : e.target.value)
          }
          className={inputCls}
        >
          {current === '' && <option value="">Choose</option>}
          {param.options.map((opt) => (
            <option key={String(opt)} value={String(opt)}>
              {String(opt)}
            </option>
          ))}
        </select>
      </label>
    );
  }

  if (param.type === 'number') {
    const current =
      typeof value === 'number' ? value : typeof param.default === 'number' ? param.default : (param.min ?? 0);
    const bounded = param.min !== undefined && param.max !== undefined;
    // Wide 0–1 style ranges feel right as sliders; unit counts as steppers.
    const slider = bounded && param.max! - param.min! <= 6 && !Number.isInteger(param.max! - param.min!);
    return (
      <label className="flex flex-col gap-1.5 py-1">
        <span className="flex items-center justify-between text-[12px] text-os-muted" title={param.description}>
          <span>{label}</span>
          <span className="font-mono tabular-nums text-os-dim">{current}</span>
        </span>
        {slider ? (
          <input
            type="range"
            min={param.min}
            max={param.max}
            step={(param.max! - param.min!) / 20}
            value={current}
            onChange={(e) => onChange(param.name, Number(e.target.value))}
            className="w-full accent-[var(--accent)]"
          />
        ) : (
          <input
            type="number"
            min={param.min}
            max={param.max}
            value={current}
            onChange={(e) => onChange(param.name, Number(e.target.value))}
            className={inputCls}
          />
        )}
      </label>
    );
  }

  if (param.type === 'string_array') {
    const current = Array.isArray(value) ? value.join(', ') : '';
    return (
      <label className="flex flex-col gap-1.5 py-1">
        <span className="text-[12px] text-os-muted" title={param.description}>
          {label}
        </span>
        <input
          value={current}
          onChange={(e) =>
            onChange(
              param.name,
              e.target.value
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean),
            )
          }
          placeholder={param.description}
          className={inputCls}
        />
      </label>
    );
  }

  const current = typeof value === 'string' ? value : typeof param.default === 'string' ? param.default : '';
  return (
    <label className="flex flex-col gap-1.5 py-1">
      <span className="text-[12px] text-os-muted" title={param.description}>
        {label}
      </span>
      <input
        value={current}
        onChange={(e) => onChange(param.name, e.target.value)}
        placeholder={param.description}
        className={inputCls}
      />
    </label>
  );
}

export function ParamFields({
  model,
  values,
  onChange,
}: {
  model: HfModel;
  values: ParamValues;
  onChange: (name: string, value: string | number | boolean | string[]) => void;
}) {
  const { primary, advanced } = visibleParams(model);
  const [showAdvanced, setShowAdvanced] = useState(false);

  if (primary.length === 0 && advanced.length === 0) {
    return <p className="py-1 text-[12px] text-os-dim">This model takes no extra parameters.</p>;
  }

  return (
    <div className="flex flex-col gap-1">
      {primary.map((p) => (
        <Field key={p.name} param={p} value={values[p.name]} onChange={onChange} />
      ))}
      {advanced.length > 0 && (
        <div className="mt-1 border-t border-[var(--hairline)] pt-2">
          <button
            type="button"
            onClick={() => setShowAdvanced((v) => !v)}
            className="flex items-center gap-1.5 text-[12px] text-os-dim transition-colors hover:text-os-text"
          >
            <ChevronDown
              className={`h-3.5 w-3.5 transition-transform ${showAdvanced ? 'rotate-180' : ''}`}
              strokeWidth={1.7}
            />
            Advanced
          </button>
          {showAdvanced && (
            <div className="mt-1 flex flex-col gap-1">
              {advanced.map((p) => (
                <Field key={p.name} param={p} value={values[p.name]} onChange={onChange} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
