import { expect, type Page, test } from '@playwright/test';

type SmokeUser = {
  email: string;
  name: string;
  password: string;
  bio: string;
};

function uniqueSeed(): string {
  return `${Date.now()}-${Math.floor(Math.random() * 100_000)}`;
}

function makeSmokeUser(prefix: string, seed: string): SmokeUser {
  return {
    email: `${prefix}-${seed}@example.com`,
    name: `${prefix} ${seed}`,
    password: 'SmokePass123!',
    bio: `${prefix} user created by the Astro browser smoke path.`,
  };
}

async function fillManagedUserForm(page: Page, user: SmokeUser) {
  await page.getByLabel('Name').fill(user.name);
  await page.getByLabel('Email').fill(user.email);
  await page.getByLabel('Password required').fill(user.password);
  await page.getByLabel('Confirm password').fill(user.password);
  await page.getByLabel('Bio optional').fill(user.bio);
}

test('Astro browser flow works against the real Go backend', async ({
  page,
}) => {
  const seed = uniqueSeed();
  const operator = makeSmokeUser('operator', seed);
  const managed = makeSmokeUser('managed', seed);
  const updatedManagedName = `managed-updated ${seed}`;
  const updatedManagedBio = 'Updated through the Astro browser smoke path.';

  await page.goto('/');

  await expect(
    page.getByRole('heading', {
      name: /go-web-server frontend migration/i,
    }),
  ).toBeVisible();
  await expect(page.getByText(/Session-aware home page/i)).toBeVisible();
  await expect(page.getByText(/Backend health/i)).toBeVisible();
  await expect(page.getByText(/Status:/)).toBeVisible();

  await page.getByRole('link', { name: 'Create account' }).click();
  await expect(page.locator('h1', { hasText: 'Create account' })).toBeVisible();

  await page.getByLabel('Name').fill(operator.name);
  await page.getByLabel('Email').fill(operator.email);
  await page.getByLabel('Password', { exact: true }).fill(operator.password);
  await page
    .getByLabel('Confirm password', { exact: true })
    .fill(operator.password);
  await page.getByLabel('Bio optional').fill(operator.bio);
  await page.getByRole('button', { name: 'Create account' }).click();

  await page.waitForURL('**/profile?auth_notice=register');
  await expect(page.getByText('Registration successful.')).toBeVisible();
  await expect(page.getByText(operator.email)).toBeVisible();
  await expect(page.getByText(operator.name)).toBeVisible();

  await page.getByRole('link', { name: 'Manage users' }).click();

  await page.waitForURL('**/users');
  await expect(
    page.getByRole('heading', {
      name: 'Manage users',
    }),
  ).toBeVisible();
  await expect(page.getByText('1 active user')).toBeVisible();
  await expect(
    page.getByText(`${operator.name} (${operator.email})`),
  ).toBeVisible();

  await fillManagedUserForm(page, managed);
  await page.getByRole('button', { name: 'Create user' }).click();

  await expect(page.getByText('User created successfully')).toBeVisible();
  await expect(page.getByText('2 active users')).toBeVisible();

  const managedCard = page.locator('.user-card').filter({
    hasText: managed.email,
  });
  await expect(managedCard).toContainText(managed.name);
  await expect(managedCard).toContainText(managed.bio);

  await managedCard.getByRole('button', { name: 'Edit' }).click();

  const nameField = page.getByLabel('Name');
  const bioField = page.getByLabel('Bio optional');
  await expect(nameField).toHaveValue(managed.name);
  await expect(bioField).toHaveValue(managed.bio);

  await nameField.fill(updatedManagedName);
  await bioField.fill(updatedManagedBio);
  await page.getByRole('button', { name: 'Save changes' }).click();

  await expect(page.getByText('User updated successfully')).toBeVisible();
  await expect(
    page.locator('.user-card').filter({ hasText: managed.email }),
  ).toContainText(updatedManagedName);
  await expect(
    page.locator('.user-card').filter({ hasText: managed.email }),
  ).toContainText(updatedManagedBio);

  await page.goto('/auth/logout');
  await page.waitForURL('**/auth/login?logged_out=1');
  await expect(page.getByText('You have been logged out.')).toBeVisible();
});
