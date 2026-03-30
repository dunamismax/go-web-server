import { expect, type Page, type Route, test } from '@playwright/test';

type SessionState = {
  authenticated: boolean;
  user: null | {
    id: number;
    email: string;
    name: string;
    is_active: boolean;
  };
  csrfToken?: string;
};

function jsonHeaders(csrfToken?: string) {
  return {
    'content-type': 'application/json',
    ...(csrfToken ? { 'x-csrf-token': csrfToken } : {}),
  };
}

async function fulfillAuthState(route: Route, state: SessionState) {
  await route.fulfill({
    status: 200,
    headers: jsonHeaders(state.csrfToken ?? 'csrf-token'),
    body: JSON.stringify({
      authenticated: state.authenticated,
      user: state.user,
      csrf: {
        header: 'X-CSRF-Token',
        form_field: 'csrf_token',
        token: state.csrfToken ?? 'csrf-token',
      },
    }),
  });
}

async function mockHomeBackend(page: Page) {
  await page.route('**/_backend/health', async (route) => {
    await route.fulfill({
      status: 200,
      headers: jsonHeaders(),
      body: JSON.stringify({
        status: 'ok',
        timestamp: '2026-03-30T23:00:00Z',
        service: 'go-web-server',
        version: '4.0.0',
        uptime: '5m0s',
        checks: {
          database: 'ok',
          memory: 'ok',
        },
      }),
    });
  });

  await page.route('**/_backend/api/auth/state', async (route) => {
    await fulfillAuthState(route, {
      authenticated: false,
      user: null,
    });
  });
}

test('renders the Astro home page with session-aware actions', async ({
  page,
}) => {
  await mockHomeBackend(page);

  await page.goto('/');

  await expect(
    page.getByRole('heading', {
      name: /go-web-server frontend migration/i,
    }),
  ).toBeVisible();
  await expect(page.getByText(/Session-aware home page/i)).toBeVisible();
  await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible();
  await expect(
    page.getByRole('link', { name: 'Create account' }),
  ).toBeVisible();
  await expect(page.getByText(/Status:/)).toBeVisible();
});

test('login success redirects into the Astro profile page', async ({
  page,
}) => {
  let authenticated = false;

  await page.route('**/_backend/api/auth/state', async (route) => {
    await fulfillAuthState(route, {
      authenticated,
      user: authenticated
        ? {
            id: 7,
            email: 'user@example.com',
            name: 'Example User',
            is_active: true,
          }
        : null,
      csrfToken: authenticated ? 'csrf-after-login' : 'csrf-before-login',
    });
  });

  await page.route('**/_backend/api/auth/login', async (route) => {
    authenticated = true;

    await route.fulfill({
      status: 200,
      headers: jsonHeaders('csrf-after-login'),
      body: JSON.stringify({
        message: 'Login successful',
        user: {
          id: 7,
          email: 'user@example.com',
          name: 'Example User',
          is_active: true,
        },
      }),
    });
  });

  await page.goto('/auth/login');
  await page.getByLabel('Email').fill('user@example.com');
  await page.getByLabel('Password').fill('Password1');
  await page.getByRole('button', { name: 'Sign in' }).click();

  await page.waitForURL('**/profile?auth_notice=login');
  await expect(page.getByText('Login successful.')).toBeVisible();
  await expect(page.getByText('Example User')).toBeVisible();
  await expect(page.getByText('user@example.com')).toBeVisible();
});

test('unauthenticated profile access redirects to the Astro login page', async ({
  page,
}) => {
  await page.route('**/_backend/api/auth/state', async (route) => {
    await fulfillAuthState(route, {
      authenticated: false,
      user: null,
    });
  });

  await page.goto('/profile');

  await page.waitForURL(
    '**/auth/login?return_to=%2Fprofile&reason=auth-required',
  );
  await expect(page.getByText('Please sign in to continue.')).toBeVisible();
});

test('logout returns the user to the Astro login page', async ({ page }) => {
  let loggedOut = false;

  await page.route('**/_backend/api/auth/state', async (route) => {
    await fulfillAuthState(route, {
      authenticated: !loggedOut,
      user: loggedOut
        ? null
        : {
            id: 7,
            email: 'user@example.com',
            name: 'Example User',
            is_active: true,
          },
      csrfToken: loggedOut ? 'csrf-login' : 'csrf-logout',
    });
  });

  await page.route('**/_backend/api/auth/logout', async (route) => {
    loggedOut = true;

    await route.fulfill({
      status: 200,
      headers: jsonHeaders('csrf-login'),
      body: JSON.stringify({
        message: 'Logout successful',
      }),
    });
  });

  await page.goto('/auth/logout');

  await page.waitForURL('**/auth/login?logged_out=1');
  await expect(page.getByText('You have been logged out.')).toBeVisible();
});
