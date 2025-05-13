import kernel
from kernel import Kernel
from playwright.async_api import async_playwright
from typing import TypedDict

client = Kernel()

# Create a new Kernel app
app = kernel.App("python-basic")

class PageTitleInput(TypedDict):
    url: str

class PageTitleOutput(TypedDict):
    title: str

@app.action("get-page-title")
async def get_page_title(ctx: kernel.KernelContext, input_data: PageTitleInput) -> PageTitleOutput:
    """
    A function that extracts the title of a webpage
    
    Args:
        ctx: Kernel context containing invocation information
        input_data: An object with a URL property
        
    Returns:
        A dictionary containing the page title
    """
    url = input_data.get("url")
    if not url or not isinstance(url, str):
        raise ValueError("URL is required and must be a string")
    
    # Create a browser instance using the context's invocation_id
    kernel_browser = client.browser.create_session(invocation_id=ctx.invocation_id)
    
    async with async_playwright() as playwright:
        browser = await playwright.chromium.connect_over_cdp(kernel_browser.cdp_ws_url)
        context = await browser.new_context()
        page = await context.new_page()
        
        try:
            await page.goto(url)
            title = await page.title()
            
            return {"title": title}
        finally:
            await browser.close()
