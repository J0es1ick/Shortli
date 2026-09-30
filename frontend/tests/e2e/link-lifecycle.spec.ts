import { randomUUID } from "node:crypto";
import { expect, test } from "@playwright/test";

test("registration, QR editing, pause/resume, deletion and session revocation", async ({
  page,
  request,
}) => {
  const suffix = randomUUID().slice(0, 12);
  const email = `e2e-${suffix}@example.com`;
  const password = `Test-password-42-${suffix}`;
  const code = `e2e-${suffix}`;
  try {
    await page.goto("/register");
    await page.getByLabel("Email", { exact: true }).fill(email);
    await page.getByLabel("Password", { exact: true }).fill(password);
    await page.getByLabel("Confirm password", { exact: true }).fill(password);
    await page.getByRole("button", { name: "Create account" }).click();
    await expect(
      page.getByRole("button", { name: new RegExp(`e2e-${suffix}.*Log out`) }),
    ).toBeVisible();
    await page
      .getByLabel("Paste a long URL")
      .fill(`https://example.com/${code}`);
    await page.getByLabel("Custom ending").fill(code);
    await page.getByRole("button", { name: "Shorten", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "Customize QR code" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Customize QR code" }).click();
    const qr = page.getByRole("dialog");
    const canvas = qr.locator("canvas");
    await expect
      .poll(() => canvas.evaluate((element) => element.width))
      .toBeGreaterThan(0);
    const before = await canvas.evaluate((element) => element.toDataURL());
    await qr.getByRole("button", { name: "Cobalt" }).click();
    await expect
      .poll(() => canvas.evaluate((element) => element.toDataURL()))
      .not.toBe(before);
    for (let step = 0; step < 24; step++) {
      await page.keyboard.press("Tab");
      expect(
        await qr.evaluate((element) =>
          element.contains(document.activeElement),
        ),
      ).toBe(true);
    }
    await page.keyboard.press("Escape");
    await expect(qr).toHaveCount(0);
    const row = page
      .locator("article")
      .filter({ hasText: `https://example.com/${code}` });
    await row.getByRole("button", { name: "Manage", exact: true }).click();
    const details = page.getByRole("dialog");
    await details.getByRole("button", { name: "Pause", exact: true }).click();
    await expect(details).toHaveCount(0);
    await row.getByRole("button", { name: "Manage", exact: true }).click();
    await expect(
      details.getByRole("button", { name: "Resume", exact: true }),
    ).toBeVisible();
    expect((await request.get(`/${code}`, { maxRedirects: 0 })).status()).toBe(
      410,
    );
    await details.getByRole("button", { name: "Resume", exact: true }).click();
    await expect(details).toHaveCount(0);
    await row.getByRole("button", { name: "Manage", exact: true }).click();
    await expect(
      details.getByRole("button", { name: "Pause", exact: true }),
    ).toBeVisible();
    expect((await request.get(`/${code}`, { maxRedirects: 0 })).status()).toBe(
      302,
    );
    await page.keyboard.press("Escape");
    await expect(
      row.getByRole("button", { name: "Manage", exact: true }),
    ).toBeFocused();
    await row.getByRole("button", { name: "Share", exact: true }).click();
    const share = page.getByRole("dialog");
    for (let step = 0; step < 18; step++) {
      await page.keyboard.press("Shift+Tab");
      expect(
        await share.evaluate((element) =>
          element.contains(document.activeElement),
        ),
      ).toBe(true);
    }
    await page.keyboard.press("Escape");
    await row.getByRole("button", { name: "Delete", exact: true }).click();
    await row.getByRole("button", { name: "Confirm", exact: true }).click();
    await expect(row).toHaveCount(0);
    expect((await request.get(`/${code}`, { maxRedirects: 0 })).status()).toBe(
      404,
    );
    await page
      .getByRole("button", { name: new RegExp(`e2e-${suffix}.*Log out`) })
      .click();
    await page
      .getByRole("alertdialog")
      .getByRole("button", { name: "Log out", exact: true })
      .click();
    await expect(
      page
        .getByRole("banner")
        .getByRole("button", { name: "Sign in", exact: true }),
    ).toBeVisible();
    expect((await page.request.get("/api/me")).status()).toBe(401);
    await page
      .getByRole("banner")
      .getByRole("button", { name: "Sign in", exact: true })
      .click();
    await page
      .getByRole("dialog")
      .getByLabel("Email", { exact: true })
      .fill(email);
    await page
      .getByRole("dialog")
      .getByLabel("Password", { exact: true })
      .fill(password);
    await page
      .getByRole("dialog")
      .getByRole("button", { name: /Sign in/ })
      .click();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: new RegExp(`e2e-${suffix}.*Log out`) }),
    ).toBeVisible();
  } finally {
    const login = await page.request.post("/api/login", {
      data: { email, password },
    });
    if (login.ok()) {
      const cookie = login.headers()["set-cookie"].split(";")[0];
      const cleanup = await page.request.delete("/api/user/account", {
        headers: { Cookie: cookie },
      });
      expect
        .soft(
          cleanup.ok(),
          `Test account cleanup failed: HTTP ${cleanup.status()}`,
        )
        .toBe(true);
    }
  }
});
