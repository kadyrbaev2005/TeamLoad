import { expect, test, type Browser, type Page } from "@playwright/test";

// Happy path (brief §10): register → create project → approve plan → second
// user joins and fills in the profile → run matching → accept all → add task
// → dashboard. Matching needs three members, so a third teammate joins too.

const run = Date.now();
const description =
  "A web app for SDU students to find and book free group-study rooms in the library. Students see a map of rooms with live availability, book a 1-2 hour slot and get a QR code to check in at the door.";

function watchConsole(page: Page, errors: string[]) {
  page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
  page.on("pageerror", (e) => errors.push(e.message));
}

async function register(page: Page, name: string, email: string) {
  await page.getByRole("tab", { name: "Create account" }).click();
  await page.getByLabel("Full name").fill(name);
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("studyspace26");
  await page.locator("#fConsent").check();
  await page.getByRole("button", { name: "Create account" }).click();
}

async function fillProfile(page: Page, strengths: string, weaknesses: string) {
  await page.getByLabel("Strengths: what are you good at?").fill(strengths);
  await page.getByLabel("Weaknesses: what is hard for you?").fill(weaknesses);
  await page.getByRole("button", { name: "Save profile" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Profile saved" })).toBeVisible();
}

async function teammate(browser: Browser, link: string, name: string, email: string, strengths: string, weaknesses: string, errors: string[]) {
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  watchConsole(page, errors);
  await page.goto(link);
  await expect(page).toHaveURL(/\/login\?next=/);
  await register(page, name, email);
  await expect(page.getByRole("heading", { name: "StudySpace E2E" })).toBeVisible();
  await expect(page.getByText(/\d \/ 3/)).toBeVisible();
  await page.getByRole("button", { name: "Join StudySpace E2E" }).click();
  await expect(page).toHaveURL(/\/profile$/);
  await expect(page.getByText("You joined StudySpace E2E. Fill in your profile next.")).toBeVisible();
  await fillProfile(page, strengths, weaknesses);
  await ctx.close();
}

test("owner and teammates go from sign-up to the workload dashboard", async ({ page, browser }) => {
  const errors: string[] = [];
  watchConsole(page, errors);

  // Owner registers.
  await page.goto("/");
  await expect(page).toHaveURL(/\/login/);
  await register(page, "Aruzhan Sadykova", `aruzhan.${run}@stu.sdu.edu.kz`);
  await expect(page).toHaveURL(/\/projects$/);

  // Creates a project; the AI designs the roles behind the overlay.
  await page.getByRole("link", { name: "New project", exact: true }).click();
  await page.getByLabel("Project title").fill("StudySpace E2E");
  await page.getByLabel("What are you building?").fill(description);
  await page.getByLabel("Deadline").fill("2026-12-11");
  await page.getByRole("button", { name: "Fewer members" }).click();
  await page.getByRole("button", { name: "Fewer members" }).click();
  await page.getByRole("button", { name: "Generate role plan" }).click();
  await expect(page.getByRole("heading", { name: "Designing your team" })).toBeVisible();
  await expect(page).toHaveURL(/\/p\/[^/]+\/plan$/);
  await expect(page.getByRole("heading", { name: "3 roles for StudySpace E2E" })).toBeVisible();

  // Approves the plan.
  await page.getByRole("button", { name: "Approve plan" }).click();
  await expect(page.getByText("Plan approved. Matching is now available.")).toBeVisible();
  const code = (await page.locator(".code").textContent())?.trim() ?? "";
  expect(code).toMatch(/^[A-HJ-NP-Z2-9]{6}$/);

  // Owner fills in the profile.
  await page.getByRole("link", { name: "My profile" }).click();
  await fillProfile(page,
    "I organise group work well and keep deadlines. I like talking to the instructor and presenting our progress.",
    "Not strong at writing backend code.");

  // Second and third teammates join with the invite link.
  const link = `/join/${code}`;
  await teammate(browser, link, "Daniyar Kenzhebekov", `daniyar.${run}@stu.sdu.edu.kz`,
    "Go and PostgreSQL. I built a REST API during my internship and I like algorithms and clean data models.",
    "I don't enjoy CSS or visual design.", errors);
  await teammate(browser, link, "Timur Akhmetov", `timur.${run}@stu.sdu.edu.kz`,
    "React and TypeScript. I made two front-end pet projects and I care about fast, responsive pages.",
    "I hate writing documentation.", errors);

  // Owner runs matching and accepts all.
  await page.getByRole("link", { name: "Team roles" }).click();
  await expect(page.getByRole("heading", { name: "Everyone is ready." })).toBeVisible();
  await page.getByRole("button", { name: "Run AI matching" }).click();
  await expect(page.getByRole("heading", { name: "Matching people to roles" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Who does what" })).toBeVisible();
  await expect(page.locator(".match")).toHaveCount(3);
  await page.getByRole("button", { name: "Accept all" }).click();
  await expect(page.getByText("All roles confirmed")).toBeVisible();
  await expect(page.getByText("3 of 3 roles confirmed.")).toBeVisible();

  // Adds a task.
  await page.getByRole("link", { name: /^Board/ }).click();
  await page.getByRole("button", { name: "New task" }).click();
  await page.getByLabel("Title").fill("Booking API: create and cancel");
  await page.getByLabel("Estimate (hours)").fill("12");
  await page.getByLabel("Assignee").selectOption({ label: "Daniyar Kenzhebekov" });
  await page.getByRole("dialog").getByRole("button", { name: "Add task" }).click();
  await expect(page.getByText("Task added")).toBeVisible();
  await expect(page.locator(".card", { hasText: "Booking API: create and cancel" })).toContainText("12 h");

  // Dashboard shows Daniyar overloaded (12 h vs a 4 h average).
  await page.getByRole("link", { name: "Dashboard" }).click();
  await expect(page.getByRole("heading", { name: "Workload" })).toBeVisible();
  await expect(page.getByText("Daniyar is overloaded.")).toBeVisible();
  await expect(page.getByRole("img", { name: "Active hours per member compared to the team average" })).toBeVisible();

  expect(errors).toEqual([]);
});
