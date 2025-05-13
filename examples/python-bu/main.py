from langchain_openai import ChatOpenAI
from browser_use import Agent, Browser, BrowserConfig
import kernel
from kernel import Kernel
from typing import TypedDict
import os

client = Kernel()

app = kernel.App("python-bu")

os.environ["OPENAI_API_KEY"] = "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"

llm = ChatOpenAI(model="gpt-4o")

class TaskInput(TypedDict):
    task: str

@app.action("bu-task")
async def bu_task(ctx: kernel.KernelContext, input_data: TaskInput):
    kernel_browser = client.browser.create_session(invocation_id=ctx.invocation_id)
    print("kernel browser remote view url: ", kernel_browser.remote_url)
    agent = Agent(
        #task="Compare the price of gpt-4o and DeepSeek-V3",
        task=input_data["task"],
        llm=llm,
        browser=Browser(BrowserConfig(cdp_url=kernel_browser.cdp_ws_url))
    )
    result = await agent.run()
    if result.final_result() is not None:
      return {"final_result": result.final_result()}
    return {"errors": result.errors()}
