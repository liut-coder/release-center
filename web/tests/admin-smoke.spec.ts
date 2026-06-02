import { expect, type Locator, type Page, test } from "@playwright/test";

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
    const userRow = page.getByRole("row").filter({ has: page.getByRole("cell", { name: "新用户" }) }).first();
    await expect(userRow).toBeVisible();
    await toggleTableRowState(page, userRow, "停用", "启用");

    await submitSystemManagementForms(page);

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

async function submitSystemManagementForms(page: Page) {
  await openSidebarIfMobile(page);
  await navButton(page, "角色管理").click();
  await expect(page.getByRole("heading", { name: "角色管理" })).toBeVisible();
  await page.getByRole("button", { name: "新增角色" }).click();
  const roleCard = page.getByTestId(/^system-role-row-custom_role_/).first();
  await expect(roleCard.getByText("新角色")).toBeVisible();
  await toggleRoleCardState(page, roleCard);
  await expectNoPageOverflow(page);

  await openSidebarIfMobile(page);
  await navButton(page, "权限管理").click();
  await expect(page.getByRole("heading", { name: "权限管理" })).toBeVisible();
  await page.getByRole("button", { name: "新增权限" }).click();
  const permissionRow = page.getByRole("row").filter({ has: page.getByRole("cell", { name: "新权限" }) }).first();
  await expect(permissionRow).toBeVisible();
  await toggleTableRowState(page, permissionRow, "停用", "启用");
  await expectNoPageOverflow(page);

  await openSidebarIfMobile(page);
  await navButton(page, "数据字典").click();
  await expect(page.getByRole("heading", { name: "数据字典" })).toBeVisible();
  await page.getByRole("button", { name: "新增字典" }).click();
  const dictionaryRow = page.getByRole("row").filter({ has: page.getByRole("cell", { name: "新字典项" }) }).first();
  await expect(dictionaryRow).toBeVisible();
  await toggleTableRowState(page, dictionaryRow, "停用", "启用");
  await expectNoPageOverflow(page);

  await openSidebarIfMobile(page);
  await navButton(page, "菜单编辑").click();
  await expect(page.getByRole("heading", { name: "菜单编辑" })).toBeVisible();
  await page.getByRole("button", { name: "新增菜单" }).click();
  const menuRow = page.getByRole("row").filter({ has: page.getByRole("cell", { name: "新菜单" }) }).first();
  await expect(menuRow).toBeVisible();
  await toggleTableRowState(page, menuRow, "隐藏", "显示");
  await expectNoPageOverflow(page);
}

async function toggleTableRowState(page: Page, row: Locator, firstAction: string, secondAction: string) {
  await row.getByRole("button", { name: firstAction, exact: true }).click();
  await expect(row.getByRole("button", { name: secondAction, exact: true })).toBeVisible();
  await row.getByRole("button", { name: secondAction, exact: true }).click();
  await expect(row.getByRole("button", { name: firstAction, exact: true })).toBeVisible();
  await expectNoPageOverflow(page);
}

async function toggleRoleCardState(page: Page, roleCard: Locator) {
  await roleCard.getByRole("switch").click();
  await expect(roleCard.getByText("停用")).toBeVisible();
  await roleCard.getByRole("switch").click();
  await expect(roleCard.getByText("启用")).toBeVisible();
  await expectNoPageOverflow(page);
}

async function submitDemoReleaseForms(page: Page) {
  const suffix = Date.now().toString().slice(-6);
  const appKey = `smoke-app-${suffix}`;
  const resourceVersion = `${todayCompact()}.9${suffix}`;
  const releaseTitle = `Smoke 发布 ${suffix}`;
  const updatedReleaseTitle = `Smoke 发布已编辑 ${suffix}`;
  const updatedResourceTitle = `Smoke 资源已编辑 ${suffix}`;

  await operateDemoAppManagement(page, appKey, suffix);

  await releaseTab(page, "构建记录").click();
  const buildVersion = await useRecommendedBuildVersion(page);
  await page.getByPlaceholder("branch / tag / commit").fill("main");
  await page.getByPlaceholder("buildNumber").fill(buildVersion.versionCode);
  await page.getByPlaceholder("构建备注").fill("Playwright demo smoke build");
  await expect(page.getByRole("button", { name: "创建构建任务" })).toBeEnabled();
  await page.getByRole("button", { name: "创建构建任务" }).click();
  await expect(page.getByText("构建任务已创建")).toBeVisible();
  await expect(page.getByText(buildVersion.versionName).first()).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseTab(page, "App 发布").click();
  await page.getByPlaceholder("标题").fill(releaseTitle);
  await page.getByPlaceholder("摘要").fill("Playwright demo release draft");
  await page.getByRole("button", { name: "创建发布草稿" }).click();
  await expect(page.getByText("发布草稿已创建")).toBeVisible();
  const releaseRow = page.getByTestId(`release-row-${buildVersion.versionName}`);
  await expect(releaseRow.getByText(releaseTitle)).toBeVisible();
  await expectNoPageOverflow(page);
  await operateDemoRelease(page, releaseRow, updatedReleaseTitle);

  await releaseTab(page, "资源增量").click();
  await page.getByPlaceholder("resourceVersion").fill(resourceVersion);
  await page.getByLabel("上传资源 ZIP").setInputFiles({
    name: `templates-common-${resourceVersion}.zip`,
    mimeType: "application/zip",
    buffer: buildZip([{ name: "templates/demo.txt", content: `demo ${suffix}` }]),
  });
  await page.getByRole("button", { name: "创建资源版本" }).click();
  await expect(page.getByText("资源版本已创建")).toBeVisible();
  const resourceRow = page.getByTestId(`resource-row-${resourceVersion}`);
  await expect(resourceRow.getByText(resourceVersion).first()).toBeVisible();
  await expectNoPageOverflow(page);
  await operateDemoResource(page, resourceRow, updatedResourceTitle);

  await verifyAuditTrail(page, buildVersion.versionName, resourceVersion);
}

async function useRecommendedBuildVersion(page: Page) {
  await page.getByRole("button", { name: "使用建议值" }).click();
  const versionNameInput = page.getByPlaceholder("versionName");
  const versionCodeInput = page.getByPlaceholder("versionCode");
  const versionName = await versionNameInput.inputValue();
  const versionCode = await versionCodeInput.inputValue();
  await expect(versionNameInput).toHaveValue(/.+/);
  await expect(versionCodeInput).toHaveValue(/^[1-9]\d*$/);
  return { versionName, versionCode };
}

async function operateDemoAppManagement(page: Page, appKey: string, suffix: string) {
  await releaseTab(page, "概览").click();
  await expect(page.getByText("App 列表")).toBeVisible();
  await page.getByPlaceholder("app_key").fill(appKey);
  await page.getByPlaceholder("应用名称").fill(`Smoke 应用 ${suffix}`);
  await page.getByPlaceholder("平台").fill("android");
  await page.getByPlaceholder("包名").fill(`com.releasecenter.smoke${suffix}`);
  await page.getByPlaceholder("描述").fill("Playwright demo app management smoke");
  await page.getByRole("button", { name: "保存应用" }).click();
  await expect(page.getByText("应用已保存")).toBeVisible();
  const appRow = page.getByTestId(`app-row-${appKey}`);
  await expect(appRow.getByText(`Smoke 应用 ${suffix}`)).toBeVisible();
  await expect(appRow.getByText("启用").first()).toBeVisible();
  await appRow.getByRole("button", { name: "停用", exact: true }).click();
  await expect(page.getByText("应用已停用")).toBeVisible();
  await expect(appRow.getByText("停用").first()).toBeVisible();
  await appRow.getByRole("button", { name: "启用", exact: true }).click();
  await expect(page.getByText("应用已启用")).toBeVisible();
  await expect(appRow.getByText("启用").first()).toBeVisible();
  await expectNoPageOverflow(page);
}

async function operateDemoRelease(page: Page, releaseRow: Locator, updatedTitle: string) {
  await releaseRow.getByRole("button", { name: "发布", exact: true }).click();
  await expect(page.getByRole("heading", { name: "发布版本" })).toBeVisible();
  await page.getByRole("button", { name: "确认发布" }).click();
  await expect(page.getByText("发布状态已更新")).toBeVisible();
  await expect(releaseRow.getByText("已发布").first()).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseRow.getByRole("button", { name: "灰度", exact: true }).click();
  await expect(page.getByRole("heading", { name: "调整 App 灰度" })).toBeVisible();
  await page.getByLabel("自定义灰度比例").fill("50");
  await page.getByRole("button", { name: "确认调整" }).click();
  await expect(page.getByText("灰度比例已同步")).toBeVisible();
  await expect(releaseRow.getByText("50%").first()).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseRow.getByRole("button", { name: "编辑说明", exact: true }).click();
  await expect(page.getByRole("heading", { name: "编辑版本说明" })).toBeVisible();
  await page.getByLabel("说明标题").fill(updatedTitle);
  await page.getByLabel("说明摘要").fill("Playwright demo release notes updated");
  await page.getByLabel("Markdown 更新内容").fill("## Smoke\n- release operation verified");
  await page.getByRole("button", { name: "保存说明" }).click();
  await expect(page.getByText("更新说明已保存")).toBeVisible();
  await expect(releaseRow.getByText(updatedTitle)).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseRow.getByRole("button", { name: "撤回", exact: true }).click();
  await expect(page.getByRole("heading", { name: "撤回版本" })).toBeVisible();
  await page.getByRole("button", { name: "确认撤回" }).click();
  await expect(page.getByText("版本已撤回")).toBeVisible();
  await expect(releaseRow.getByText("已撤回").first()).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseRow.getByRole("button", { name: "回滚", exact: true }).click();
  await expect(page.getByRole("heading", { name: "回滚版本" })).toBeVisible();
  await page.getByRole("button", { name: "确认回滚" }).click();
  await expect(page.getByText("已回滚到选定版本")).toBeVisible();
  await expect(releaseRow.getByText("已发布").first()).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseRow.getByRole("button", { name: "下架", exact: true }).click();
  await expect(page.getByRole("heading", { name: "下架版本" })).toBeVisible();
  await page.getByRole("button", { name: "确认下架" }).click();
  await expect(page.getByText("版本已下架")).toBeVisible();
  await expect(releaseRow.getByText("已撤回").first()).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseRow.getByRole("button", { name: "发布", exact: true }).click();
  await expect(page.getByRole("heading", { name: "发布版本" })).toBeVisible();
  await page.getByRole("button", { name: "确认发布" }).click();
  await expect(page.getByText("发布状态已更新")).toBeVisible();
  await expect(releaseRow.getByText("已发布").first()).toBeVisible();
  await expectNoPageOverflow(page);

  await releaseRow.getByRole("button", { name: "暂停", exact: true }).click();
  await expect(page.getByRole("heading", { name: "暂停发布" })).toBeVisible();
  await page.getByRole("button", { name: "确认暂停" }).click();
  await expect(page.getByText("发布已暂停")).toBeVisible();
  await expect(releaseRow.getByText("已暂停").first()).toBeVisible();
  await expectNoPageOverflow(page);
}

async function operateDemoResource(page: Page, resourceRow: Locator, updatedTitle: string) {
  await resourceRow.getByRole("button", { name: "发布", exact: true }).click();
  await expect(page.getByRole("heading", { name: "发布资源版本" })).toBeVisible();
  await page.getByRole("button", { name: "确认发布" }).click();
  await expect(page.getByText("资源版本已发布")).toBeVisible();
  await expect(resourceRow.getByText("已发布").first()).toBeVisible();
  await expectNoPageOverflow(page);

  await resourceRow.getByRole("button", { name: "灰度", exact: true }).click();
  await expect(page.getByRole("heading", { name: "调整资源灰度" })).toBeVisible();
  await page.getByLabel("自定义灰度比例").fill("30");
  await page.getByRole("button", { name: "确认调整" }).click();
  await expect(page.getByText("资源灰度已同步")).toBeVisible();
  await expect(resourceRow.getByText("30%").first()).toBeVisible();
  await expectNoPageOverflow(page);

  await resourceRow.getByRole("button", { name: "编辑说明", exact: true }).click();
  await expect(page.getByRole("heading", { name: "编辑资源说明" })).toBeVisible();
  await page.getByLabel("说明标题").fill(updatedTitle);
  await page.getByLabel("说明摘要").fill("Playwright demo resource notes updated");
  await page.getByLabel("Markdown 更新内容").fill("## Smoke\n- resource operation verified");
  await page.getByRole("button", { name: "保存说明" }).click();
  await expect(page.getByText("资源更新说明已保存")).toBeVisible();
  await expect(resourceRow.getByText(updatedTitle)).toBeVisible();
  await expectNoPageOverflow(page);

  await resourceRow.getByRole("button", { name: "回滚", exact: true }).click();
  await expect(page.getByRole("heading", { name: "回滚资源版本" })).toBeVisible();
  await page.getByRole("button", { name: "确认回滚" }).click();
  await expect(page.getByText("资源指针已回滚")).toBeVisible();
  await expect(resourceRow.getByText("已发布").first()).toBeVisible();
  await expectNoPageOverflow(page);

  await resourceRow.getByRole("button", { name: "暂停", exact: true }).click();
  await expect(page.getByRole("heading", { name: "暂停资源版本" })).toBeVisible();
  await page.getByRole("button", { name: "确认暂停" }).click();
  await expect(page.getByText("资源版本已暂停")).toBeVisible();
  await expect(resourceRow.getByText("已暂停").first()).toBeVisible();
  await expectNoPageOverflow(page);
}

async function verifyAuditTrail(page: Page, versionName: string, resourceVersion: string) {
  await releaseTab(page, "操作审计").click();
  await expect(page.getByRole("heading", { name: "App 发布" })).toBeVisible();
  await expect(page.getByPlaceholder("搜索操作人 / 动作 / 对象")).toBeVisible();

  await page.getByPlaceholder("搜索操作人 / 动作 / 对象").fill(versionName);
  await expect(page.getByRole("cell", { name: "创建发布" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "发布版本" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "调整灰度" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "更新版本说明" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "撤回版本" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "回滚版本" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "下架版本" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "暂停发布" }).first()).toBeVisible();
  await expectNoPageOverflow(page);

  await page.getByPlaceholder("搜索操作人 / 动作 / 对象").fill(resourceVersion);
  await expect(page.getByRole("cell", { name: "创建资源" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "发布资源" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "调整资源灰度" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "更新资源说明" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "回滚资源" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "暂停资源" }).first()).toBeVisible();
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
