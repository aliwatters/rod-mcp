# Demo Video

Create a video demo from a Playwright script in this repository.

## Instructions

### Step 1: Discover Demo Scripts

Search for demo scripts in the current repository:

To discover demo scripts in the current repository, run:

```bash
find . -name "demo*.ts" -o -name "demo*.js" | grep -E "(scripts|e2e)" | head -20
```

Common locations to check:
- `scripts/demo-*.ts`
- `e2e-tests/**/demo-*.ts`
- `tests/demo-*.ts`

For each script found, try to extract the description from JSDoc comments at the top of the file.

### Step 2: Present Options to User

List found scripts in a numbered format:

```
I found these demo scripts in this repo:

1. scripts/demo-ce-registration.ts - CE Registration flow demo
2. scripts/demo-clinic-card.ts - ClinicCard component demo

Which would you like to create a video of? (enter number)
```

If no demo scripts are found, inform the user and provide guidance on creating one:

```
No demo scripts found in this repository.

To create a demo script, create a file like `scripts/demo-your-feature.ts`
that uses Playwright to record browser interactions.

See ~/creating-playwright-videos.md for a complete guide.
```

### Step 3: Check Prerequisites

Before running the selected script, verify:

1. **Dev server is running** (check the expected port):

   > **Determine the correct port for your project:**
   > - Check `package.json` "dev" script for port number
   > - Check `.env` or `.env.local` for PORT variable
   > - Common ports: 3000 (Next.js default), 5173 (Vite), 8000

   ```bash
   curl -s -o /dev/null -w "%{http_code}" http://localhost:<PORT>
   ```
   - If not running (returns 000 or error), inform the user: "Dev server is not running on the expected port. Please start it with `pnpm dev` in another terminal."

2. **Environment variables exist**:
   - Check for `.env.local` or `.env` file
   - Warn if neither exists

3. **Playwright is installed**:
   ```bash
   pnpm list @playwright/test | grep playwright
   ```

Report any issues and ask user if they want to proceed anyway.

### Step 4: Execute the Script

Run the selected demo script:

> **Replace placeholders:**
> - `<project-root>`: The repository root directory (use `pwd` or `git rev-parse --show-toplevel`)
> - `<script-path>`: The path to the script selected in Step 2 (e.g., `scripts/demo-clinic-card.ts`)

```bash
cd <project-root> && pnpm exec tsx <script-path>
```

Monitor the execution and report progress to the user.

### Step 5: Handle Video Output

After script completion:

1. **Find the generated video** in the temp directory (usually `./videos/` or `./videos-temp/`)

2. **Move to Desktop** with a descriptive name:

   > **Replace `<script-name>` with the base name of your script (e.g., `demo-clinic-card` from `scripts/demo-clinic-card.ts`)**

   ```bash
   mv "$(ls -t ./videos/*.webm | head -1)" ~/Desktop/<script-name>-$(date +%Y%m%d-%H%M%S).webm
   ```

3. **Clean up temp directory** if empty:
   ```bash
   rmdir ./videos || true
   ```

4. **Report to user**:
   ```
   Video saved to: ~/Desktop/demo-clinic-card-20251211-143022.webm
   ```

## Video Recording Guidelines

Reference: ~/creating-playwright-videos.md

Key points for demo scripts:
- Use `headless: false` for visible recording
- Add `waitForTimeout()` pauses (2-3 seconds) so viewers can see content
- Always call `clerkSetup()` before browser launch if using Clerk auth
- Close page -> context -> browser in order to save video properly
- Structure demos in clear "scenes" with console.log messages

## Troubleshooting

### Video not saved or 0 bytes
Ensure the script closes resources in order:
```typescript
await page.close();      // First
await context.close();   // Second (triggers video save)
await browser.close();   // Last
```

### Auth not persisting (307 redirects)
Wait for sign-in to fully complete before navigating:
```typescript
await clerk.signIn({ /* ... */ });
await page.waitForSelector('[data-testid="authenticated-content"]');
// Don't navigate immediately - auth state needs to establish
```

### ERR_ABORTED errors
Usually means trying to access protected page without auth. Sign in first and wait for redirect.

## Example Demo Script Structure

```typescript
/**
 * Demo script for [Feature Name]
 *
 * Usage: pnpm exec tsx scripts/demo-feature.ts
 */
import { chromium } from "@playwright/test";
import { clerk, clerkSetup } from "@clerk/testing/playwright";
import * as dotenv from "dotenv";

dotenv.config({ path: ".env.local" });
dotenv.config({ path: ".env" });

async function main() {
  console.log("Setting up Clerk...");
  await clerkSetup();

  const browser = await chromium.launch({ headless: false });
  const context = await browser.newContext({
    recordVideo: { dir: "./videos/", size: { width: 1280, height: 720 } }
  });
  const page = await context.newPage();

  try {
    // Sign in
    await page.goto("http://localhost:8000/auth/sign-in");
    await clerk.signIn({ page, signInParams: { /* ... */ } });

    // Demo scenes here...
    await page.waitForTimeout(3000); // Pause for viewers

  } finally {
    await page.close();
    await context.close();
    await browser.close();
  }
}

main().catch(console.error);
```
