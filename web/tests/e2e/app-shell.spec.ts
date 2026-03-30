import { expect, test } from '@playwright/test';

test('renders the migration shell', async ({ page }) => {
  await page.goto('/');

  await expect(
    page.getByRole('heading', {
      name: /go-web-server frontend migration shell/i,
    }),
  ).toBeVisible();
  await expect(
    page.getByText(
      /Astro owns the page shell, Vue owns the interactive status card/i,
    ),
  ).toBeVisible();
  await expect(
    page.getByText(
      /Same-origin-friendly proxy path for local backend requests/i,
    ),
  ).toBeVisible();
});
