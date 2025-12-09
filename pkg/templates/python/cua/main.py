import os
from typing import TypedDict
import kernel
from kernel import Kernel
from computers.default import KernelPlaywrightBrowser
from agent import Agent
import datetime
import asyncio

"""
Example app that runs an agent using openai CUA
Args:
    ctx: Kernel context containing invocation information
    payload: An object with a `task` property
Returns:
    An answer to the task, elapsed time and optionally the messages stack
Invoke this via CLI:
    kernel login  # or: export KERNEL_API_KEY=<your_api_key>
    kernel deploy main.py -e OPENAI_API_KEY=XXXXX --force
    kernel invoke python-cua cua-task -p '{"task":"go to https://news.ycombinator.com and list top 5 articles"}'
    kernel logs python-cua -f # Open in separate tab
"""

class CuaInput(TypedDict):
    task: str

class CuaOutput(TypedDict):
    result: str

api_key = os.getenv("OPENAI_API_KEY")
if not api_key:
    raise ValueError("OPENAI_API_KEY is not set")

client = Kernel()
app = kernel.App("python-cua")

@app.action("cua-task")
async def cua_task(
    ctx: kernel.KernelContext,
    payload: CuaInput,
) -> CuaOutput:
    # A function that processes a user task using the kernel browser and agent

    if not payload or not payload.get("task"):
        raise ValueError("task is required")

    kernel_browser = await asyncio.to_thread(
        client.browsers.create, invocation_id=ctx.invocation_id, stealth=True
    )
    print("Kernel browser live view url: ", kernel_browser.browser_live_view_url)
    cdp_ws_url = kernel_browser.cdp_ws_url

    def run_agent():
        with KernelPlaywrightBrowser({"cdp_ws_url": cdp_ws_url}) as computer:

            # messages to provide to the agent
            items = [
                {
                    "role": "system",
                    "content": f"- Current date and time: {datetime.datetime.utcnow().isoformat()} ({datetime.datetime.utcnow().strftime('%A')})",
                },
                {
                    "role": "user",
                    "content": payload["task"]
                }
            ]

            # setup the agent
            agent = Agent(
                computer=computer,
                tools=[], # can provide additional tools to the agent
                acknowledge_safety_check_callback=lambda message: (print(f"> agent : safety check message (skipping): {message}") or True) # safety check function , now defaults to true
            )

            # run the agent
            response_items = agent.run_full_turn(
                items,
                debug=True,
                show_images=False,
            )

            if not response_items or "content" not in response_items[-1]:
                raise ValueError("No response from agent")
            # The content may be a list of blocks, get the first text block
            content = response_items[-1]["content"]
            if isinstance(content, list) and content and isinstance(content[0], dict) and "text" in content[0]:
                result = content[0]["text"]
            elif isinstance(content, str):
                result = content
            else:
                result = str(content)
            return {"result": result}

    return await asyncio.to_thread(run_agent)
