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
    await submitDemoReleaseForms(page);

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

async function submitDemoReleaseForms(page: Page) {
  const suffix = Date.now().toString().slice(-6);
  const patch = Number(suffix.slice(-2)) || 2;
  const versionName = `1.0.${patch}-dev.1`;
  const versionCode = 1_000_000_000 + Number(suffix);
  const buildNumber = versionCode;
  const resourceVersion = `${todayCompact()}.99${suffix.slice(-2)}`;

  await releaseTab(page, "构建记录").click();
  await page.getByPlaceholder("branch / tag / commit").fill("main");
  await page.getByPlaceholder("versionName").fill(versionName);
  await page.getByPlaceholder("versionCode").fill(String(versionCode));
  await page.getByPlaceholder("buildNumber").fill(String(buildNumber));
  await page.getByPlaceholder("构建备注").fill("Playwright demo smoke build");
  await page.getByRole("button", { name: "创建构建任务" }).click();
  await expect(page.getByText("构建任务已创建")).toBeVisible();
  await expect(page.getByText(versionName).first()).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseTab(page, "App 发布").click();
  await page.getByPlaceholder("标题").fill(`Smoke 发布 ${suffix}`);
  await page.getByPlaceholder("摘要").fill("Playwright demo release draft");
  await page.getByRole("button", { name: "创建发布草稿" }).click();
  await expect(page.getByText("发布草稿已创建")).toBeVisible();
  await expect(page.getByText(`Smoke 发布 ${suffix}`).first()).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseTab(page, "资源增量").click();
  await page.getByPlaceholder("resourceVersion").fill(resourceVersion);
  await page.getByLabel("上传资源 ZIP").setInputFiles({
    name: `templates-common-${resourceVersion}.zip`,
    mimeType: "application/zip",
    buffer: buildZip([{ name: "templates/demo.txt", content: `demo ${suffix}` }]),
  });
  await page.getByRole("button", { name: "创建资源版本" }).click();
  await expect(page.getByText("资源版本已创建")).toBeVisible();
  await expect(page.getByText(resourceVersion).first()).toBeVisible();
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

function todayCompact() {
  const date = new Date();
  return `${date.getFullYear()}${String(date.getMonth() + 1).padStart(2, "0")}${String(date.getDate()).padStart(2, "0")}`;
}

function buildZip(entries: Array<{ name: string; content: string }>) {
  const fileRecords: Buffer[] = [];
  const centralRecords: Buffer[] = [];
  let offset = 0;
  for (const entry of entries) {
    const name = Buffer.from(entry.name);
    const content = Buffer.from(entry.content);
    const crc = crc32(content);
    const local = Buffer.alloc(30 + name.length);
    local.writeUInt32LE(0x04034b50, 0);
    local.writeUInt16LE(20, 4);
    local.writeUInt16LE(0, 6);
    local.writeUInt16LE(0, 8);
    local.writeUInt16LE(0, 10);
    local.writeUInt16LE(0, 12);
    local.writeUInt32LE(crc, 14);
    local.writeUInt32LE(content.length, 18);
    local.writeUInt32LE(content.length, 22);
    local.writeUInt16LE(name.length, 26);
    name.copy(local, 30);
    fileRecords.push(local, content);

    const central = Buffer.alloc(46 + name.length);
    central.writeUInt32LE(0x02014b50, 0);
    central.writeUInt16LE(20, 4);
    central.writeUInt16LE(20, 6);
    central.writeUInt16LE(0, 8);
    central.writeUInt16LE(0, 10);
    central.writeUInt16LE(0, 12);
    central.writeUInt16LE(0, 14);
    central.writeUInt32LE(crc, 16);
    central.writeUInt32LE(content.length, 20);
    central.writeUInt32LE(content.length, 24);
    central.writeUInt16LE(name.length, 28);
    central.writeUInt32LE(offset, 42);
    name.copy(central, 46);
    centralRecords.push(central);
    offset += local.length + content.length;
  }
  const centralOffset = offset;
  const central = Buffer.concat(centralRecords);
  const end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50, 0);
  end.writeUInt16LE(entries.length, 8);
  end.writeUInt16LE(entries.length, 10);
  end.writeUInt32LE(central.length, 12);
  end.writeUInt32LE(centralOffset, 16);
  return Buffer.concat([...fileRecords, central, end]);
}

function crc32(input: Buffer) {
  let crc = 0xffffffff;
  for (const byte of input) {
    crc ^= byte;
    for (let i = 0; i < 8; i++) {
      crc = crc & 1 ? (crc >>> 1) ^ 0xedb88320 : crc >>> 1;
    }
  }
  return (crc ^ 0xffffffff) >>> 0;
}
