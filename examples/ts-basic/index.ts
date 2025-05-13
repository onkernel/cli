import { Kernel, type KernelContext } from '@onkernel/sdk';
import { chromium } from 'playwright';

const kernel = new Kernel();

const app = kernel.app('ts-basic');

interface PageTitleInput {
  url: string;
}

interface PageTitleOutput {
  title: string;
}

app.action<PageTitleInput, PageTitleOutput>(
  'get-page-title',
  async (ctx: KernelContext, input: PageTitleInput): Promise<PageTitleOutput> => {
    if (!input.url) {
      throw new Error('URL is required');
    }

    const kernelBrowser = await kernel.browser.createSession({
      invocationId: ctx.invocationId,
    });

    // Connect to the browser using Playwright
    const browser = await chromium.connectOverCDP(kernelBrowser.cdp_ws_url);
    const context = await browser.newContext();
    const page = await context.newPage();

    try {
      await page.goto(input.url);
      const title = await page.title();
      return { title };
    } finally {
      await browser.close();
    }
  },
);
