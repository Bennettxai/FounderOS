import { describe, it, expect } from 'vitest';
import { detectLanguage, getLanguageLabel } from './language-detection';

describe('detectLanguage', () => {
  it('detects language from common extensions', () => {
    expect(detectLanguage('app.ts')).toBe('typescript');
    expect(detectLanguage('index.js')).toBe('javascript');
    expect(detectLanguage('styles.css')).toBe('css');
    expect(detectLanguage('main.go')).toBe('go');
    expect(detectLanguage('schema.sql')).toBe('sql');
    expect(detectLanguage('config.yaml')).toBe('yaml');
    expect(detectLanguage('README.md')).toBe('markdown');
  });

  it('detects language from full file paths', () => {
    expect(detectLanguage('src/lib/api/base.ts')).toBe('typescript');
    expect(detectLanguage('internal/handler/auth.go')).toBe('go');
    expect(detectLanguage('styles/theme.scss')).toBe('scss');
  });

  it('detects language from exact filename matches', () => {
    expect(detectLanguage('Dockerfile')).toBe('dockerfile');
    expect(detectLanguage('Makefile')).toBe('plaintext');
    expect(detectLanguage('.gitignore')).toBe('plaintext');
    expect(detectLanguage('go.mod')).toBe('go');
    expect(detectLanguage('tsconfig.json')).toBe('json');
    expect(detectLanguage('package.json')).toBe('json');
  });

  it('is case-insensitive for extensions', () => {
    expect(detectLanguage('file.JS')).toBe('javascript');
    expect(detectLanguage('file.TS')).toBe('typescript');
    expect(detectLanguage('file.HTML')).toBe('html');
  });

  it('returns plaintext for unknown extensions', () => {
    expect(detectLanguage('file.xyz')).toBe('plaintext');
    expect(detectLanguage('file.unknown')).toBe('plaintext');
  });

  it('returns plaintext for empty/extensionless files', () => {
    expect(detectLanguage('')).toBe('plaintext');
    expect(detectLanguage('noextension')).toBe('plaintext');
  });
});

describe('getLanguageLabel', () => {
  it('returns display labels for known languages', () => {
    expect(getLanguageLabel('typescript')).toBe('TypeScript');
    expect(getLanguageLabel('javascript')).toBe('JavaScript');
    expect(getLanguageLabel('go')).toBe('Go');
    expect(getLanguageLabel('python')).toBe('Python');
  });

  it('returns the raw ID for unknown languages', () => {
    expect(getLanguageLabel('unknownlang')).toBe('unknownlang');
  });
});
