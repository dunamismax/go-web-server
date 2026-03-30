type ImportMetaWithEnv = ImportMeta & {
  env?: Record<string, string | undefined>;
};

const defaultBackendBase = '/_backend';

export interface HealthResponse {
  status: string;
  timestamp: string;
  service: string;
  version: string;
  uptime: string;
  checks: Record<string, string>;
}

export function normalizeBackendBase(value?: string): string {
  const trimmed = (value ?? '').trim();
  if (trimmed === '') {
    return defaultBackendBase;
  }

  const withLeadingSlash = trimmed.startsWith('/') ? trimmed : `/${trimmed}`;
  return withLeadingSlash.endsWith('/') && withLeadingSlash !== '/'
    ? withLeadingSlash.slice(0, -1)
    : withLeadingSlash;
}

export function configuredBackendBase(
  meta: ImportMetaWithEnv = import.meta as ImportMetaWithEnv,
): string {
  return normalizeBackendBase(meta.env?.PUBLIC_BACKEND_PROXY_BASE);
}

export function backendPath(
  path: string,
  base = configuredBackendBase(),
): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${normalizeBackendBase(base)}${normalizedPath}`;
}

export async function fetchHealth(
  fetcher: typeof fetch = fetch,
  base = configuredBackendBase(),
): Promise<HealthResponse> {
  const response = await fetcher(backendPath('/health', base), {
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error(`Health request failed with status ${response.status}`);
  }

  return (await response.json()) as HealthResponse;
}
