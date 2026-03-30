import { describe, expect, test } from 'bun:test';

import { backendPath, normalizeBackendBase } from './backend';

describe('normalizeBackendBase', () => {
  test('defaults to the migration proxy path', () => {
    expect(normalizeBackendBase()).toBe('/_backend');
  });

  test('adds a leading slash when needed', () => {
    expect(normalizeBackendBase('api')).toBe('/api');
  });

  test('removes a trailing slash', () => {
    expect(normalizeBackendBase('/proxy/')).toBe('/proxy');
  });
});

describe('backendPath', () => {
  test('joins the proxy base and request path', () => {
    expect(backendPath('/health', '/_backend')).toBe('/_backend/health');
  });

  test('normalizes bare paths', () => {
    expect(backendPath('demo', '/proxy')).toBe('/proxy/demo');
  });
});
