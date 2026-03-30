import { describe, expect, test } from 'bun:test';

import {
  backendPath,
  extractFieldErrors,
  normalizeBackendBase,
  setCSRFToken,
} from './backend';
import { appendQueryParams, normalizeReturnTo } from './navigation';

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

describe('extractFieldErrors', () => {
  test('reads array-based validation payloads', () => {
    expect(
      extractFieldErrors([
        { field: 'email', message: 'invalid email format' },
        { field: 'password', message: 'password is required' },
      ]),
    ).toEqual({
      email: 'invalid email format',
      password: 'password is required',
    });
  });

  test('reads object-shaped conflict payloads', () => {
    expect(
      extractFieldErrors({
        email: 'Email already exists',
      }),
    ).toEqual({
      email: 'Email already exists',
    });
  });
});

describe('normalizeReturnTo', () => {
  test('allows same-origin relative paths', () => {
    expect(normalizeReturnTo('/users?tab=active')).toBe('/users?tab=active');
  });

  test('rejects absolute urls', () => {
    expect(normalizeReturnTo('https://evil.example/phish')).toBe('/profile');
  });

  test('rejects protocol-relative paths', () => {
    expect(normalizeReturnTo('//evil.example/phish')).toBe('/profile');
  });
});

describe('appendQueryParams', () => {
  test('adds and replaces query params on relative paths', () => {
    expect(
      appendQueryParams('/auth/login?return_to=%2Fprofile', {
        logged_out: '1',
      }),
    ).toBe('/auth/login?return_to=%2Fprofile&logged_out=1');
  });

  test('drops empty params', () => {
    expect(
      appendQueryParams('/profile?auth_notice=login', {
        auth_notice: undefined,
      }),
    ).toBe('/profile');
  });
});

describe('setCSRFToken', () => {
  test('stores a trimmed token', () => {
    expect(setCSRFToken('  token  ')).toBe('token');
  });
});
