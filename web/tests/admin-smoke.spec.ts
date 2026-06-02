import { expect, type Page, test } from "@playwright/test";

test.describe("admin console smoke", () => {
  test("release admin sees authorized navigation without runtime errors", async ({ page }) => {
    const errors = collectClientErrors(page);

    await loginAs(page, "release-token", "release.admin");
    await openSidebarIfMobile(page);

    await expect(navButton(page, "发布中心")).toBeVisible();
    await expect(navButton(page, "用户管理")).toHaveCount(0);
    await expect(navButton(page, "角色管理")).toHaveCount(0);
    await expect(page.getByText("发布中心", { exact: true }).first()).toBeVisible();

    await navButton(page, "发布中心").click();
    await expectAppReleaseShell(page);
    await expect(page.getByText("正式版本")).toBeVisible();
    await verifyAppReleaseTabs(page);

    expect(errors(), "browser runtime errors").toEqual([]);
  });

  test("system admin can open system pages and create demo user", async ({ page }) => {
    const errors = collectClientErrors(page);

    await loginAs(page, "admin-token", "system.admin");
    await openSidebarIfMobile(page);

    await navButton(page, "用户管理").click();
    await expect(page.getByRole("heading", { name: "用户管理" })).toBeVisible();
    await expect(page.getByRole("cell", { name: "system.admin" })).toBeVisible();

    await page.getByRole("button", { name: "新增用户" }).click();
    await expect(page.getByRole("cell", { name: "新用户" }).first()).toBeVisible();

    await openSidebarIfMobile(page);
    await navButton(page, "菜单编辑").click();
    await expect(page.getByRole("heading", { name: "菜单编辑" })).toBeVisible();
    await expect(page.getByRole("cell", { name: "/release-center" })).toBeVisible();
    await expectNoPageOverflow(page);

    expect(errors(), "browser runtime errors").toEqual([]);
  });
});

async function loginAs(page: Page, token: string, account: string) {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "后台登录" })).toBeVisible();
  await page.getByLabel("账号").fill(account);
  await page.getByLabel("后台 Token").fill(token);
  await page.getByRole("button", { name: "登录" }).click();
  await expect(page.getByText("已认证")).toBeVisible();
  await expect(page.getByText("首页").first()).toBeVisible();
}

async function expectAppReleaseShell(page: Page) {
  await expect(page.getByRole("heading", { name: "App 发布" })).toBeVisible();
  await expect(page.getByRole("button", { name: "刷新" })).toBeVisible();
  await expectNoPageOverflow(page);
}

async function verifyAppReleaseTabs(page: Page) {
  await releaseTab(page, "构建记录").click();
  await expect(page.getByText("构建新版本")).toBeVisible();
  await expect(page.getByPlaceholder("branch / tag / commit")).toBeVisible();
  await expect(page.getByPlaceholder("apiBaseUrl")).toHaveValue(/\/$/);
  await expect(page.getByRole("button", { name: "创建构建任务" })).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseTab(page, "App 发布").click();
  await expect(page.getByText("创建 App 发布")).toBeVisible();
  await expect(page.getByRole("button", { name: "创建发布草稿" })).toBeVisible();
  await expect(page.getByPlaceholder("搜索版本 / commit / 文件名")).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseTab(page, "资源增量").click();
  await expect(page.getByText("创建资源增量发布")).toBeVisible();
  await expect(page.getByPlaceholder("resourceVersion")).toBeVisible();
  await expect(page.getByRole("button", { name: "创建资源版本" })).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseTab(page, "设备版本").click();
  await expect(page.getByPlaceholder("搜索设备 / 用户 / App 版本 / 资源版本")).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseTab(page, "升级统计").click();
  await expect(page.getByText("升级概况")).toBeVisible();
  await expect(page.getByText("事件分布")).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseTab(page, "升级事件").click();
  await expect(page.getByPlaceholder("搜索设备 / 事件 / 资源包 / 错误")).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseTab(page, "操作审计").click();
  await expect(page.getByPlaceholder("搜索操作人 / 动作 / 对象")).toBeVisible();
  await expectNoPageOverflow(page);
}

function releaseTab(page: Page, name: string) {
  return page.getByRole("button", { name, exact: true });
}

function navButton(page: Page, name: string) {
  return page.getByRole("navigation").getByRole("button", { name, exact: true });
}

async function expectNoPageOverflow(page: Page) {
  const overflow = await page.evaluate(() => {
    const root = document.documentElement;
    return Math.max(0, root.scrollWidth - root.clientWidth);
  });
  expect(overflow, "page horizontal overflow").toBeLessThanOrEqual(2);
}

async function openSidebarIfMobile(page: Page) {
  const viewport = page.viewportSize();
  if (viewport && viewport.width < 1024) {
    const button = page.getByRole("button", { name: "打开菜单" });
    if (await button.isVisible()) {
      await button.click();
    }
  }
}

function collectClientErrors(page: Page) {
  const errors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") {
      errors.push(`console.error: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => {
    errors.push(`pageerror: ${error.message}`);
  });
  page.on("requestfailed", (request) => {
    const url = request.url();
    if (url.includes("/admin/api/") || url.includes("/api/")) {
      errors.push(`requestfailed: ${request.method()} ${url} ${request.failure()?.errorText ?? ""}`.trim());
    }
  });
  page.on("response", (response) => {
    const url = response.url();
    const status = response.status();
    if ((url.includes("/admin/api/") || url.includes("/api/")) && status >= 500) {
      errors.push(`response: ${response.request().method()} ${url} ${status}`);
    }
  });
  return () => errors;
}
