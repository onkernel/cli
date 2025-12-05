import { Stagehand } from "@browserbasehq/stagehand";
import { Kernel, type KernelContext } from '@onkernel/sdk';

const kernel = new Kernel({
  apiKey: process.env.KERNEL_API_KEY
});

const app = kernel.app('ts-gemini-cua');

interface SearchQueryOutput {
  success: boolean;
  result: string;
  error?: string;
}

// API Keys for LLM providers
// - GOOGLE_API_KEY: Required for Gemini 2.5 Computer Use Agent
// - OPENAI_API_KEY: Required for Stagehand's GPT-4o model
// Set via environment variables or `kernel deploy <filename> --env-file .env`
// See https://docs.onkernel.com/launch/deploy#environment-variables
const GOOGLE_API_KEY = process.env.GOOGLE_API_KEY;
const OPENAI_API_KEY = process.env.OPENAI_API_KEY;

if (!OPENAI_API_KEY) {
  throw new Error('OPENAI_API_KEY is not set');
}

if (!GOOGLE_API_KEY) {
  throw new Error('GOOGLE_API_KEY is not set');
}

async function runStagehandTask(invocationId?: string): Promise<SearchQueryOutput> {
  // Executes a Computer Use Agent (CUA) task using Gemini 2.5 and Stagehand

  const browserOptions = {
    stealth: true,
    viewport: {
      width: 1440,
      height: 900,
      refresh_rate: 25
    },
    ...(invocationId && { invocation_id: invocationId })
  };

  const kernelBrowser = await kernel.browsers.create(browserOptions);

  console.log("Kernel browser live view url: ", kernelBrowser.browser_live_view_url);

  const stagehand = new Stagehand({
    env: "LOCAL",
    verbose: 1,
    domSettleTimeoutMs: 30_000,
    modelName: "gpt-4o",
    modelClientOptions: {
      apiKey: OPENAI_API_KEY
    },
    localBrowserLaunchOptions: {
      cdpUrl: kernelBrowser.cdp_ws_url
    }
  });
  await stagehand.init();

  /////////////////////////////////////
  // Your Stagehand implementation here
  /////////////////////////////////////
  try {
    const page = stagehand.page;

    const agent = stagehand.agent({
      provider: "google",
      model: "gemini-2.5-computer-use-preview-10-2025",
      instructions: `You are a helpful assistant that can use a web browser.
      You are currently on the following page: ${page.url()}.
      Do not ask follow up questions, the user will trust your judgement.`,
      options: {
        apiKey: GOOGLE_API_KEY,
      }
    });

    // Navigate to YCombinator's website
    await page.goto("https://www.ycombinator.com/companies");

    // Define the instructions for the CUA agent
    const instruction = "Find Kernel's company page on the YCombinator website and write a blog post about their product offering.";

    // Execute the instruction
    const result = await agent.execute({
      instruction,
      maxSteps: 20,
    });

    console.log("result: ", result);

    return { success: true, result: result.message };
  } catch (error) {
    console.error(error);
    const errorMessage = error instanceof Error ? error.message : String(error);
    return { success: false, result: "", error: errorMessage };
  } finally {
    console.log("Deleting browser and closing stagehand...");
    await stagehand.close();
    await kernel.browsers.deleteByID(kernelBrowser.session_id);
  }
}

// Register Kernel action handler for remote invocation
// Invoked via: kernel invoke ts-gemini-cua gemini-cua-task
app.action<void, SearchQueryOutput>(
  'gemini-cua-task',
  async (ctx: KernelContext): Promise<SearchQueryOutput> => {
    return runStagehandTask(ctx.invocation_id);
  },
);

// Run locally if executed directly (not imported as a module)
// Execute via: npx tsx index.ts
if (import.meta.url === `file://${process.argv[1]}`) {
  runStagehandTask().then(result => {
    console.log('Local execution result:', result);
    process.exit(result.success ? 0 : 1);
  }).catch(error => {
    console.error('Local execution failed:', error);
    process.exit(1);
  });
}
