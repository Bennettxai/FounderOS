import type { CSSProperties, ElementType, ReactNode } from 'react';

/**
 * Slab motion, OS-wide. The brand-deals slab's entry beat as a shared
 * wrapper: the Legacy kit's l-rise (app/globals.css), 10px and a hair of
 * scale on the house ease, staggered by its index at 90ms a step. Transform
 * only, never a fade (glass brief §4: a fading parent kills the frost of the
 * faces inside it). Server-safe: a class and a CSS variable. Reduced motion
 * lands instantly (the L0.6 guard in globals.css).
 *
 * <Rise i={2} className="glass-face gl-materialise l-rise-card"> a glass card
 *   that materialises on the stagger and lifts on hover (SlabCard)
 * <Rise as="section" i={4} className="mb-5">
 */
export function Rise({
  i = 0,
  as,
  className = '',
  style,
  children,
  ...rest
}: {
  i?: number;
  as?: ElementType;
  className?: string;
  style?: CSSProperties;
  children?: ReactNode;
} & Record<string, unknown>) {
  const Tag = (as ?? 'div') as ElementType;
  const vars = { ...style, '--l-rise-i': i } as CSSProperties;
  return (
    <Tag className={`l-rise ${className}`.trim()} style={vars} {...rest}>
      {children}
    </Tag>
  );
}
