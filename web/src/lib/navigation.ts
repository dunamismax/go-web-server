const fallbackOrigin = 'https://frontend.local';

export function normalizeReturnTo(
  value?: string | null,
  fallback = '/profile',
): string {
  const trimmed = (value ?? '').trim();
  if (trimmed === '') {
    return fallback;
  }

  if (!trimmed.startsWith('/')) {
    return fallback;
  }

  if (trimmed.startsWith('//')) {
    return fallback;
  }

  return trimmed;
}

export function appendQueryParams(
  path: string,
  params: Record<string, string | undefined>,
): string {
  const url = new URL(path, fallbackOrigin);

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === '') {
      url.searchParams.delete(key);
      continue;
    }

    url.searchParams.set(key, value);
  }

  return `${url.pathname}${url.search}${url.hash}`;
}

export function redirectTo(path: string, replace = false): void {
  if (typeof window === 'undefined') {
    return;
  }

  if (replace) {
    window.location.replace(path);
    return;
  }

  window.location.assign(path);
}
