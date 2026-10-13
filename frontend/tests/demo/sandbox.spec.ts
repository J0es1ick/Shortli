import { expect, test as base, type Page } from "@playwright/test";

const test = base.extend<{ networkGuard: void }>({
  networkGuard: [
    async ({ page }, use) => {
      const unexpected: string[] = [];
      const errors: string[] = [];
      page.on("request", (request) => {
        const url = new URL(request.url());
        if (
          ["http:", "https:"].includes(url.protocol) &&
          (url.origin !== "http://127.0.0.1:43179" ||
            !url.pathname.startsWith("/Shortli/") ||
            url.pathname.includes("/api/"))
        )
          unexpected.push(url.href);
      });
      page.on("pageerror", (error) => errors.push(error.message));
      await use();
      expect(
        unexpected,
        "The sandbox must never contact an API or external service",
      ).toEqual([]);
      expect(errors).toEqual([]);
    },
    { auto: true },
  ],
});

const go = (page: Page, path = "/") => page.goto(`./#${path}`);
const chooseRole = async (page: Page, role: string) => {
  await page.getByLabel("Try a role").selectOption(role);
  await expect(page.getByLabel("Try a role")).toBeEnabled();
  await expect(page.getByLabel("Try a role")).toHaveValue(role);
};
const createLink = async (page: Page, code: string) => {
  await page.getByLabel("Paste a long URL").fill(`https://example.com/${code}`);
  await page.getByLabel("Custom ending").fill(code);
  await page.getByRole("button", { name: "Shorten", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Customize QR code" }),
  ).toBeVisible();
};

test("registration, login and logout are local and never store passwords", async ({
  page,
}) => {
  await go(page, "/register");
  await expect(page.getByRole("note")).toContainText("not real authentication");
  await page.getByLabel("Email", { exact: true }).fill("visitor@example.com");
  await page.getByLabel("Password", { exact: true }).fill("demo-pass-123");
  await page
    .getByLabel("Confirm password", { exact: true })
    .fill("demo-pass-123");
  await page.getByRole("button", { name: "Create account" }).click();
  const account = page.getByRole("button", { name: /visitor.*Log out/ });
  await expect(account).toBeVisible();
  const storage = await page.evaluate(() => JSON.stringify(localStorage));
  expect(storage).toContain("visitor@example.com");
  expect(storage).not.toContain("demo-pass-123");
  await page.reload();
  await account.click();
  await page
    .getByRole("alertdialog")
    .getByRole("button", { name: "Log out", exact: true })
    .click();
  await page
    .getByRole("banner")
    .getByRole("button", { name: "Sign in", exact: true })
    .click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("Email", { exact: true }).fill("visitor@example.com");
  await dialog
    .getByLabel("Password", { exact: true })
    .fill("different-demo-password");
  await dialog.getByRole("button", { name: /Sign in/ }).click();
  await expect(dialog).toHaveCount(0);
  await expect(account).toBeVisible();
});

test("links, QR, deduplication, analytics, pause, expiry and deletion work without a backend", async ({
  page,
}) => {
  await go(page);
  await chooseRole(page, "user");
  await page.screenshot({ path: "test-results/demo-desktop.png" });
  await createLink(page, "sandbox-link");
  const short = page.locator('#shorten a[href$="#/s/sandbox-link"]');
  await expect(short).toHaveAttribute(
    "href",
    "http://127.0.0.1:43179/Shortli/#/s/sandbox-link",
  );
  await page
    .getByLabel("Paste a long URL")
    .fill("https://example.com/sandbox-link");
  await page.getByLabel("Custom ending").fill("duplicate-alias");
  await page.getByRole("button", { name: "Shorten", exact: true }).click();
  await expect(short).toBeVisible();
  const row = page
    .locator("article")
    .filter({ hasText: "https://example.com/sandbox-link" });
  await expect(row).toHaveCount(1);
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
  await page.keyboard.press("Escape");
  await go(page, "/s/sandbox-link");
  await expect(page.getByRole("status")).toContainText("Demo visits: 0");
  await page.getByRole("button", { name: "Simulate a visit" }).click();
  await expect(page.getByRole("status")).toContainText("Demo visits: 1");
  await expect(page).toHaveURL(/#\/s\/sandbox-link$/);
  await go(page);
  await row.getByRole("button", { name: "Manage", exact: true }).click();
  const details = page.getByRole("dialog");
  await expect(details.getByLabel("Clicks by day")).toBeVisible();
  await details.getByRole("button", { name: "Pause", exact: true }).click();
  await go(page, "/s/sandbox-link");
  await expect(page.getByRole("alert")).toContainText("paused");
  await go(page);
  await row.getByRole("button", { name: "Manage", exact: true }).click();
  await details.getByRole("button", { name: "Resume", exact: true }).click();
  await row.getByRole("button", { name: "Delete", exact: true }).click();
  await row.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect(row).toHaveCount(0);
  await go(page, "/s/sandbox-link");
  await expect(page.getByRole("alert")).toContainText("deleted");
  await go(page, "/s/newsletter");
  await expect(page.getByRole("alert")).toContainText("expired");
});

test("owner can change roles; support can moderate but cannot manage users", async ({
  page,
}) => {
  await go(page);
  await chooseRole(page, "owner");
  const member = page
    .locator("article")
    .filter({ hasText: "user@shortli.example" });
  await member.getByRole("combobox").selectOption("admin");
  await member.getByRole("button", { name: "Save role" }).click();
  await page.reload();
  await expect(member.getByRole("combobox")).toHaveValue("admin");
  await chooseRole(page, "support");
  await expect(
    page.getByRole("link", { name: "Team and roles", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Pause link", exact: true }).click();
  await expect(
    page.getByText("No pending reports.", { exact: true }),
  ).toBeVisible();
  await go(page, "/s/event");
  await expect(page.getByRole("alert")).toContainText("paused");
  await page.getByRole("button", { name: "Reset demo", exact: true }).click();
  await page.getByRole("button", { name: "Yes, reset", exact: true }).click();
  await expect(page.getByLabel("Try a role")).toHaveValue("guest");
  await chooseRole(page, "owner");
  await expect(member.getByRole("combobox")).toHaveValue("user");
  await expect(
    page.getByRole("button", { name: "Pause link", exact: true }),
  ).toBeVisible();
});

test("API keys, reports and status are clearly simulated", async ({ page }) => {
  await go(page);
  await chooseRole(page, "user");
  await go(page, "/developers");
  await expect(page.getByRole("note")).toContainText(
    "does not run an HTTP API",
  );
  await page.getByLabel("Key name").fill("Sandbox integration");
  await page.getByRole("button", { name: "Create key" }).click();
  await expect(page.getByRole("status")).toContainText("demo_not_a_real_key_");
  await page.getByRole("button", { name: "Revoke", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Revoke", exact: true }),
  ).toHaveCount(0);
  await go(page, "/report");
  await page.getByLabel("Short link or code").fill("portfolio");
  await page.getByLabel("What happened").fill("This is only a sandbox report.");
  await page.getByRole("button", { name: "Send report" }).click();
  await expect(
    page.getByRole("heading", { name: "Report received." }),
  ).toBeVisible();
  await chooseRole(page, "admin");
  await expect(
    page.getByText("This is only a sandbox report.", { exact: true }),
  ).toBeVisible();
  await go(page, "/status");
  await expect(page.getByRole("note")).toContainText("not a live server");
});

test("hash routes and section anchors survive refresh under a repository path", async ({
  page,
}) => {
  await go(page);
  await chooseRole(page, "user");
  await page
    .getByRole("navigation", { name: "Primary navigation" })
    .getByRole("link", { name: "Archive" })
    .click();
  await expect(page).toHaveURL(/\/Shortli\/#\/#history$/);
  await expect
    .poll(() =>
      page
        .locator("#history")
        .evaluate((element) => Math.abs(element.getBoundingClientRect().top)),
    )
    .toBeLessThan(200);
  await page.getByRole("button", { name: "Next →", exact: true }).click();
  await expect(
    page.locator("article").filter({ hasText: "demo-11" }),
  ).toBeVisible();
  await go(page, "/privacy");
  await page.reload();
  await page.locator("main aside a").nth(1).click();
  await expect(page).toHaveURL(/#\/privacy#section-2$/);
  await expect(page.getByRole("note")).toContainText("full product");
  const assets = await page
    .locator('script[src], link[rel="stylesheet"]')
    .evaluateAll((elements) =>
      elements.map((el) => el.getAttribute("src") || el.getAttribute("href")),
    );
  expect(
    assets.every(
      (asset) =>
        asset &&
        new URL(asset, page.url()).pathname.startsWith("/Shortli/assets/"),
    ),
  ).toBe(true);
});

test("sandbox data is isolated between visitors and persists on refresh", async ({
  page,
  browser,
}) => {
  await go(page);
  await createLink(page, "private-to-browser");
  await page.reload();
  await go(page, "/s/private-to-browser");
  await expect(page.locator("main code")).toHaveText(
    "https://example.com/private-to-browser",
  );
  const other = await browser.newContext({ locale: "en-US" });
  try {
    const visitor = await other.newPage();
    await visitor.goto("http://127.0.0.1:43179/Shortli/#/s/private-to-browser");
    await expect(visitor.getByRole("alert")).toContainText(
      "only exists in the browser",
    );
  } finally {
    await other.close();
  }
});

test("Russian mobile sandbox remains usable", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.addInitScript(() => localStorage.setItem("shortli:locale", "ru"));
  await go(page);
  await expect(page.getByLabel("Попробовать роль")).toHaveValue("guest");
  await page.getByText("Как это работает", { exact: true }).click();
  await expect(page.getByText(/Статистика синтетическая/)).toBeVisible();
  await expect
    .poll(() =>
      page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    )
    .toBe(true);
  await page.screenshot({
    path: "test-results/demo-mobile.png",
    fullPage: true,
  });
});
