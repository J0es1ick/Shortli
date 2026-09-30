import { expect, test } from "@playwright/test";

const member = {
  user_id: 1,
  email: "browser-test@example.com",
  role: "user",
  is_admin: false,
};

test("login traps focus in both directions and restores the opener", async ({
  page,
}) => {
  await page.route("**/api/me", (route) =>
    route.fulfill({ status: 401, json: { error: "Not authenticated" } }),
  );
  await page.goto("/");
  const opener = page
    .getByRole("banner")
    .getByRole("button", { name: "Sign in", exact: true });
  await opener.click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByLabel("Email", { exact: true })).toBeFocused();
  for (const key of ["Tab", "Shift+Tab"]) {
    for (let step = 0; step < 12; step++) {
      await page.keyboard.press(key);
      expect(
        await dialog.evaluate((element) =>
          element.contains(document.activeElement),
        ),
      ).toBe(true);
    }
  }
  expect(await page.locator("#root").evaluate((element) => element.inert)).toBe(
    true,
  );
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(opener).toBeFocused();
  expect(await page.locator("#root").evaluate((element) => element.inert)).toBe(
    false,
  );
});

for (const failure of ["http", "network", "timeout", "invalid response"]) {
  test(`logout ${failure} failure preserves the account and permits retry`, async ({
    page,
  }) => {
    await page.route("**/api/me", (route) => route.fulfill({ json: member }));
    await page.route("**/api/history?*", (route) =>
      route.fulfill({ json: { data: [], meta: { total: 0, totalPages: 1 } } }),
    );
    await page.route("**/api/logout", async (route) => {
      if (failure === "network") await route.abort("failed");
      else if (failure === "invalid response")
        await route.fulfill({
          status: 200,
          contentType: "text/html",
          body: "upstream error",
        });
      else if (failure === "http")
        await route.fulfill({
          status: 503,
          json: { error: "Failed to end session. Please try again." },
        });
      else
        await new Promise<void>((resolve) =>
          page.once("close", () => resolve()),
        );
    });
    await page.goto("/");
    const account = page.getByRole("button", { name: /browser-test.*Log out/ });
    await account.click();
    const dialog = page.getByRole("alertdialog");
    await dialog.getByRole("button", { name: "Log out", exact: true }).click();
    await expect(dialog.getByRole("alert")).toBeVisible({ timeout: 20_000 });
    await expect(
      dialog.getByRole("button", { name: "Log out", exact: true }),
    ).toBeEnabled();
    await expect(
      page.locator("header").getByText("browser-test", { exact: true }),
    ).toBeVisible();
    for (let step = 0; step < 5; step++) {
      await page.keyboard.press("Tab");
      expect(
        await dialog.evaluate((element) =>
          element.contains(document.activeElement),
        ),
      ).toBe(true);
    }
    await page.unroute("**/api/logout");
    await page.route("**/api/logout", (route) =>
      route.fulfill({ json: { status: "success" } }),
    );
    await dialog.getByRole("button", { name: "Log out", exact: true }).click();
    await expect(dialog).toHaveCount(0);
    await expect(
      page
        .getByRole("banner")
        .getByRole("button", { name: "Sign in", exact: true }),
    ).toBeVisible();
  });
}

test("session lookup outage is not presented as signed out", async ({
  page,
}) => {
  await page.route("**/api/me", (route) =>
    route.fulfill({
      status: 503,
      json: { error: "Authentication temporarily unavailable" },
    }),
  );
  await page.goto("/");
  await expect(
    page.getByRole("button", { name: "Retry session check" }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Sign in", exact: true }),
  ).toHaveCount(0);
  await page.unroute("**/api/me");
  await page.route("**/api/me", (route) => route.fulfill({ json: member }));
  await page.getByRole("button", { name: "Retry session check" }).click();
  await expect(
    page.getByRole("button", { name: /browser-test.*Log out/ }),
  ).toBeVisible();
});

test("an old session lookup cannot undo a newer login", async ({ page }) => {
  let releaseLookup!: () => void;
  const waitForRelease = new Promise<void>((resolve) => {
    releaseLookup = resolve;
  });
  await page.route("**/api/me", async (route) => {
    await waitForRelease;
    await route.fulfill({ status: 401, json: { error: "Not authenticated" } });
  });
  await page.route("**/api/register", (route) =>
    route.fulfill({ status: 201, json: member }),
  );
  await page.route("**/api/login", (route) => route.fulfill({ json: member }));
  await page.route("**/api/history?*", (route) =>
    route.fulfill({ json: { data: [], meta: { total: 0, totalPages: 1 } } }),
  );
  const lookupStarted = page.waitForRequest("**/api/me");
  await page.goto("/register");
  await lookupStarted;
  await page.getByLabel("Email", { exact: true }).fill(member.email);
  await page.getByLabel("Password", { exact: true }).fill("Test-password-42");
  await page
    .getByLabel("Confirm password", { exact: true })
    .fill("Test-password-42");
  await page.getByRole("button", { name: "Create account" }).click();
  const account = page.getByRole("button", { name: /browser-test.*Log out/ });
  await expect(account).toBeVisible();
  const response = page.waitForResponse("**/api/me");
  releaseLookup();
  await response;
  await expect(account).toBeVisible();
});
