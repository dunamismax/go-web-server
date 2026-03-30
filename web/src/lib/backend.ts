const defaultBackendBase = '/_backend';
const csrfHeaderName = 'X-CSRF-Token';

let csrfToken = '';

export interface HealthResponse {
  status: string;
  timestamp: string;
  service: string;
  version: string;
  uptime: string;
  checks: Record<string, string>;
}

export interface SessionUser {
  id: number;
  email: string;
  name: string;
  is_active: boolean;
}

export interface CSRFContract {
  header: string;
  form_field: string;
  token: string;
}

export interface AuthStateResponse {
  authenticated: boolean;
  user: SessionUser | null;
  csrf: CSRFContract;
}

export interface AuthMutationResponse {
  message: string;
  user?: SessionUser;
}

export interface LoginPayload {
  email: string;
  password: string;
}

export interface RegisterPayload {
  email: string;
  name: string;
  password: string;
  confirm_password: string;
  bio?: string;
  avatar_url?: string;
}

export interface ValidationErrorDetail {
  field?: string;
  message?: string;
  tag?: string;
}

export interface APIErrorResponse {
  type?: string;
  error?: string;
  message?: string;
  details?: unknown;
  code?: number;
  path?: string;
  method?: string;
  request_id?: string;
  timestamp?: string;
}

export class BackendError extends Error {
  readonly status: number;
  readonly type?: string;
  readonly details?: unknown;
  readonly fieldErrors: Record<string, string>;

  constructor(status: number, payload: APIErrorResponse = {}) {
    super(
      payload.message ??
        payload.error ??
        `Request failed with status ${status}`,
    );
    this.name = 'BackendError';
    this.status = status;
    this.type = payload.type;
    this.details = payload.details;
    this.fieldErrors = extractFieldErrors(payload.details);
  }
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

export function configuredBackendBase(): string {
  return normalizeBackendBase(import.meta.env.PUBLIC_BACKEND_PROXY_BASE);
}

export function backendPath(
  path: string,
  base = configuredBackendBase(),
): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${normalizeBackendBase(base)}${normalizedPath}`;
}

export function currentCSRFToken(): string {
  return csrfToken;
}

export function setCSRFToken(token?: string | null): string {
  csrfToken = (token ?? '').trim();
  return csrfToken;
}

export function extractFieldErrors(details: unknown): Record<string, string> {
  if (Array.isArray(details)) {
    return details.reduce<Record<string, string>>((accumulator, item) => {
      if (!item || typeof item !== 'object') {
        return accumulator;
      }

      const detail = item as ValidationErrorDetail;
      if (detail.field && detail.message) {
        accumulator[detail.field] = detail.message;
      }

      return accumulator;
    }, {});
  }

  if (details && typeof details === 'object') {
    return Object.entries(details).reduce<Record<string, string>>(
      (accumulator, [field, message]) => {
        if (typeof message === 'string' && message.trim() !== '') {
          accumulator[field] = message;
        }

        return accumulator;
      },
      {},
    );
  }

  return {};
}

function updateCSRFTokenFromResponse(response: Response): string {
  const token = response.headers.get(csrfHeaderName);
  if (token && token.trim() !== '') {
    csrfToken = token;
  }

  return csrfToken;
}

async function parseJSONPayload<T>(response: Response): Promise<T | undefined> {
  const contentType = response.headers.get('content-type') ?? '';
  if (!contentType.toLowerCase().includes('application/json')) {
    return undefined;
  }

  return (await response.json()) as T;
}

function buildHeaders(headers?: HeadersInit, includeJSONBody = false): Headers {
  const built = new Headers(headers);
  built.set('Accept', 'application/json');

  if (includeJSONBody) {
    built.set('Content-Type', 'application/json');
  }

  if (csrfToken !== '') {
    built.set(csrfHeaderName, csrfToken);
  }

  return built;
}

async function requestJSON<T>(
  path: string,
  init: RequestInit = {},
  base = configuredBackendBase(),
): Promise<T> {
  const method = init.method ?? 'GET';
  const hasBody = init.body !== undefined && init.body !== null;
  const response = await fetch(backendPath(path, base), {
    ...init,
    credentials: 'include',
    headers: buildHeaders(init.headers, hasBody),
  });

  const payload = await parseJSONPayload<T | APIErrorResponse>(response);
  updateCSRFTokenFromResponse(response);

  if (!response.ok) {
    throw new BackendError(
      response.status,
      (payload as APIErrorResponse | undefined) ?? {},
    );
  }

  if (payload === undefined) {
    throw new Error(`Expected JSON response from ${method} ${path}`);
  }

  return payload as T;
}

function cleanOptionalString(value?: string): string | undefined {
  const trimmed = value?.trim() ?? '';
  return trimmed === '' ? undefined : trimmed;
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

export async function fetchAuthState(
  base = configuredBackendBase(),
): Promise<AuthStateResponse> {
  const payload = await requestJSON<AuthStateResponse>(
    '/api/auth/state',
    undefined,
    base,
  );
  setCSRFToken(payload.csrf.token);
  return payload;
}

export async function login(
  payload: LoginPayload,
  base = configuredBackendBase(),
): Promise<AuthMutationResponse> {
  return requestJSON<AuthMutationResponse>(
    '/api/auth/login',
    {
      method: 'POST',
      body: JSON.stringify({
        email: payload.email.trim(),
        password: payload.password,
      }),
    },
    base,
  );
}

export async function register(
  payload: RegisterPayload,
  base = configuredBackendBase(),
): Promise<AuthMutationResponse> {
  return requestJSON<AuthMutationResponse>(
    '/api/auth/register',
    {
      method: 'POST',
      body: JSON.stringify({
        email: payload.email.trim(),
        name: payload.name.trim(),
        password: payload.password,
        confirm_password: payload.confirm_password,
        bio: cleanOptionalString(payload.bio),
        avatar_url: cleanOptionalString(payload.avatar_url),
      }),
    },
    base,
  );
}

export async function logout(
  base = configuredBackendBase(),
): Promise<AuthMutationResponse> {
  return requestJSON<AuthMutationResponse>(
    '/api/auth/logout',
    {
      method: 'POST',
      body: JSON.stringify({}),
    },
    base,
  );
}
