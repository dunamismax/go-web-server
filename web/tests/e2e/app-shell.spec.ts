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

type ManagedUserRecord = {
  id: number;
  email: string;
  name: string;
  avatar_url: string | null;
  bio: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
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

function activeUsers(users: ManagedUserRecord[]) {
  return users.filter((user) => user.is_active);
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

test('renders the Astro users dashboard from the explicit JSON contracts', async ({
  page,
}) => {
  const managedUsers: ManagedUserRecord[] = [
    {
      id: 21,
      email: 'person@example.com',
      name: 'Person Example',
      avatar_url: 'https://example.com/avatar.png',
      bio: 'First managed user in the Astro CRUD path.',
      is_active: true,
      created_at: '2026-03-30T23:00:00Z',
      updated_at: '2026-03-30T23:15:00Z',
    },
  ];

  await page.route('**/_backend/api/auth/state', async (route) => {
    await fulfillAuthState(route, {
      authenticated: true,
      user: {
        id: 7,
        email: 'admin@example.com',
        name: 'Admin User',
        is_active: true,
      },
      csrfToken: 'csrf-users',
    });
  });

  await page.route('**/_backend/api/users**', async (route) => {
    const url = new URL(route.request().url());

    if (url.pathname.endsWith('/api/users/count')) {
      await route.fulfill({
        status: 200,
        headers: jsonHeaders('csrf-users'),
        body: JSON.stringify({
          count: activeUsers(managedUsers).length,
        }),
      });
      return;
    }

    if (url.pathname.endsWith('/api/users')) {
      await route.fulfill({
        status: 200,
        headers: jsonHeaders('csrf-users'),
        body: JSON.stringify({
          users: activeUsers(managedUsers),
          count: activeUsers(managedUsers).length,
        }),
      });
      return;
    }

    throw new Error(`Unhandled users route: ${url.pathname}`);
  });

  await page.goto('/users');

  await expect(
    page.getByRole('heading', { name: 'Manage users' }),
  ).toBeVisible();
  await expect(page.getByText('Admin User (admin@example.com)')).toBeVisible();
  await expect(page.getByText('1 active user')).toBeVisible();
  await expect(page.getByText('Person Example')).toBeVisible();
  await expect(
    page.getByText('First managed user in the Astro CRUD path.'),
  ).toBeVisible();
  await expect(page.getByRole('link', { name: 'Open avatar' })).toBeVisible();
});

test('Astro users dashboard supports create, edit, deactivate, and delete flows', async ({
  page,
}) => {
  const managedUsers: ManagedUserRecord[] = [
    {
      id: 21,
      email: 'person@example.com',
      name: 'Person Example',
      avatar_url: null,
      bio: 'First managed user in the Astro CRUD path.',
      is_active: true,
      created_at: '2026-03-30T23:00:00Z',
      updated_at: '2026-03-30T23:15:00Z',
    },
  ];
  let nextUserID = 22;

  await page.route('**/_backend/api/auth/state', async (route) => {
    await fulfillAuthState(route, {
      authenticated: true,
      user: {
        id: 7,
        email: 'admin@example.com',
        name: 'Admin User',
        is_active: true,
      },
      csrfToken: 'csrf-users',
    });
  });

  await page.route('**/_backend/api/users**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method();

    if (url.pathname.endsWith('/api/users/count')) {
      await route.fulfill({
        status: 200,
        headers: jsonHeaders('csrf-users'),
        body: JSON.stringify({
          count: activeUsers(managedUsers).length,
        }),
      });
      return;
    }

    if (url.pathname.endsWith('/api/users')) {
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          headers: jsonHeaders('csrf-users'),
          body: JSON.stringify({
            users: activeUsers(managedUsers),
            count: activeUsers(managedUsers).length,
          }),
        });
        return;
      }

      if (method === 'POST') {
        const body = JSON.parse(request.postData() ?? '{}');
        const timestamp = new Date('2026-03-31T00:00:00Z').toISOString();
        const createdUser: ManagedUserRecord = {
          id: nextUserID,
          email: body.email,
          name: body.name,
          avatar_url: body.avatar_url ?? null,
          bio: body.bio ?? null,
          is_active: true,
          created_at: timestamp,
          updated_at: timestamp,
        };
        nextUserID += 1;
        managedUsers.push(createdUser);

        await route.fulfill({
          status: 201,
          headers: jsonHeaders('csrf-users'),
          body: JSON.stringify({
            message: 'User created successfully',
            user: createdUser,
          }),
        });
        return;
      }
    }

    const deactivateMatch = url.pathname.match(
      /\/api\/users\/(\d+)\/deactivate$/,
    );
    if (deactivateMatch && method === 'PATCH') {
      const id = Number(deactivateMatch[1]);
      const user = managedUsers.find((candidate) => candidate.id === id);
      if (!user) {
        await route.fulfill({
          status: 404,
          headers: jsonHeaders('csrf-users'),
          body: JSON.stringify({ message: 'User not found' }),
        });
        return;
      }

      user.is_active = false;
      user.updated_at = new Date('2026-03-31T00:10:00Z').toISOString();

      await route.fulfill({
        status: 200,
        headers: jsonHeaders('csrf-users'),
        body: JSON.stringify({
          message: 'User deactivated successfully',
          user,
        }),
      });
      return;
    }

    const userMatch = url.pathname.match(/\/api\/users\/(\d+)$/);
    if (userMatch) {
      const id = Number(userMatch[1]);
      const user = managedUsers.find((candidate) => candidate.id === id);

      if (method === 'GET') {
        await route.fulfill({
          status: user ? 200 : 404,
          headers: jsonHeaders('csrf-users'),
          body: JSON.stringify(
            user
              ? { user }
              : {
                  message: 'User not found',
                },
          ),
        });
        return;
      }

      if (method === 'PUT') {
        if (!user) {
          await route.fulfill({
            status: 404,
            headers: jsonHeaders('csrf-users'),
            body: JSON.stringify({ message: 'User not found' }),
          });
          return;
        }

        const body = JSON.parse(request.postData() ?? '{}');
        user.email = body.email;
        user.name = body.name;
        user.bio = body.bio ?? null;
        user.avatar_url = body.avatar_url ?? null;
        user.updated_at = new Date('2026-03-31T00:05:00Z').toISOString();

        await route.fulfill({
          status: 200,
          headers: jsonHeaders('csrf-users'),
          body: JSON.stringify({
            message: 'User updated successfully',
            user,
          }),
        });
        return;
      }

      if (method === 'DELETE') {
        const index = managedUsers.findIndex(
          (candidate) => candidate.id === id,
        );
        if (index === -1) {
          await route.fulfill({
            status: 404,
            headers: jsonHeaders('csrf-users'),
            body: JSON.stringify({ message: 'User not found' }),
          });
          return;
        }

        managedUsers.splice(index, 1);
        await route.fulfill({
          status: 200,
          headers: jsonHeaders('csrf-users'),
          body: JSON.stringify({
            id,
            deleted: true,
            message: 'User deleted successfully',
          }),
        });
        return;
      }
    }

    throw new Error(`Unhandled users route: ${method} ${url.pathname}`);
  });

  await page.goto('/users');

  await page.getByLabel('Name').fill('Second User');
  await page.getByLabel('Email').fill('second@example.com');
  await page.getByLabel('Password required').fill('Password1');
  await page.getByLabel('Confirm password').fill('Password1');
  await page
    .getByLabel('Bio optional')
    .fill('Created through the Astro users form.');
  await page.getByRole('button', { name: 'Create user' }).click();

  await expect(page.getByText('User created successfully')).toBeVisible();
  await expect(page.getByText('2 active users')).toBeVisible();
  await expect(page.getByText('Second User')).toBeVisible();

  await page
    .locator('.user-card')
    .filter({ hasText: 'Second User' })
    .getByRole('button', { name: 'Edit' })
    .click();

  const nameField = page.getByLabel('Name');
  const bioField = page.getByLabel('Bio optional');

  await expect(nameField).toHaveValue('Second User');
  await expect(bioField).toHaveValue('Created through the Astro users form.');

  await nameField.fill('Renamed User');
  await bioField.fill('Updated through the Astro users form.');
  await expect(nameField).toHaveValue('Renamed User');
  await expect(bioField).toHaveValue('Updated through the Astro users form.');

  await page.getByRole('button', { name: 'Save changes' }).click();

  await expect(page.getByText('User updated successfully')).toBeVisible();
  await expect(page.getByText('Renamed User')).toBeVisible();
  await expect(
    page.getByText('Updated through the Astro users form.'),
  ).toBeVisible();

  page.once('dialog', async (dialog) => {
    await dialog.accept();
  });
  await page
    .locator('.user-card')
    .filter({ hasText: 'Renamed User' })
    .getByRole('button', { name: 'Deactivate' })
    .click();

  await expect(page.getByText('User deactivated successfully')).toBeVisible();
  await expect(page.getByText('1 active user')).toBeVisible();
  await expect(
    page.locator('.user-card').filter({ hasText: 'Renamed User' }),
  ).toHaveCount(0);

  const firstUserCard = page
    .locator('.user-card')
    .filter({ hasText: 'Person Example' });
  page.once('dialog', async (dialog) => {
    await dialog.accept();
  });
  await firstUserCard.getByRole('button', { name: 'Delete' }).click();

  await expect(page.getByText('User deleted successfully')).toBeVisible();
  await expect(page.getByText('0 active users')).toBeVisible();
  await expect(page.getByText('No active users yet')).toBeVisible();
});
