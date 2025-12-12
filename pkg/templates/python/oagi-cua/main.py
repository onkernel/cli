import os
import sys
from types import ModuleType
from importlib.machinery import ModuleSpec

# Mock pyautogui and mouseinfo to prevent X11 connection at import time.
# oagi imports these modules internally, but we use KernelActionHandler instead,
# so pyautogui functionality is never actually needed.
mock_mouseinfo = ModuleType("mouseinfo")
mock_mouseinfo.__spec__ = ModuleSpec("mouseinfo", None)

mock_pyautogui = ModuleType("pyautogui")
mock_pyautogui.__spec__ = ModuleSpec("pyautogui", None)

sys.modules["mouseinfo"] = mock_mouseinfo
sys.modules["pyautogui"] = mock_pyautogui

# Set OAGI API base URL (no .env file in deployed environment)
os.environ.setdefault("OAGI_BASE_URL", "https://api.agiopen.org")

from typing import TypedDict, List, Optional
from kernel import App, KernelContext

from kernel_session import KernelBrowserSession
from kernel_provider import KernelScreenshotProvider
from kernel_handler import KernelActionHandler

from oagi import AsyncDefaultAgent, TaskerAgent

"""
Example app that runs agents using OpenAGI's Lux computer-use models.

Two actions are available:
1. oagi-default-task: Uses AsyncDefaultAgent for high-level tasks
2. oagi-tasker-task: Uses TaskerAgent for structured workflows with predefined steps

Args:
    ctx: Kernel context containing invocation information
    payload: Task-specific input parameters

Invoke via CLI:
    kernel login  # or: export KERNEL_API_KEY=<your_api_key>
    kernel deploy main.py -e OAGI_API_KEY=XXXXX --force

    # AsyncDefaultAgent example:
    kernel invoke python-oagi-cua oagi-default-task -p '{"instruction":"Navigate to https://agiopen.org"}'

    # TaskerAgent example:
    kernel invoke python-oagi-cua oagi-tasker-task -p '{"task":"Navigate to OAGI homepage","todos":["Go to https://agiopen.org","Click on What is Computer Use"]}'
"""


class DefaultAgentInput(TypedDict):
    instruction: str
    model: Optional[str]


class TaskerAgentInput(TypedDict):
    task: str
    todos: List[str]


class AgentOutput(TypedDict):
    success: bool
    result: str


api_key = os.getenv("OAGI_API_KEY")
if not api_key:
    raise ValueError("OAGI_API_KEY is not set")

app = App("python-oagi-cua")


@app.action("oagi-default-task")
async def oagi_default_task(
    ctx: KernelContext,
    payload: DefaultAgentInput,
) -> AgentOutput:
    """
    Execute a task using OpenAGI's AsyncDefaultAgent.

    Args:
        ctx: Kernel context containing invocation information
        payload: Contains 'instruction' (str) and optional 'model' (str, default: "lux-actor-1")

    Returns:
        AgentOutput with success status and result message
    """
    if not payload or not payload.get("instruction"):
        raise ValueError("instruction is required")

    instruction = payload["instruction"]
    model = payload.get("model", "lux-actor-1")

    async with KernelBrowserSession(record_replay=False) as session:
        print("Kernel browser live view url:", session.live_view_url)

        provider = KernelScreenshotProvider(session)
        handler = KernelActionHandler(session)

        agent = AsyncDefaultAgent(
            api_key=api_key,
            max_steps=20,
            model=model,
        )

        print(f"\nExecuting task: {instruction}\n")
        success = await agent.execute(
            instruction=instruction,
            action_handler=handler,
            image_provider=provider,
        )

        return {
            "success": success,
            "result": f"Task completed with model {model}. Success: {success}",
        }


@app.action("oagi-tasker-task")
async def oagi_tasker_task(
    ctx: KernelContext,
    payload: TaskerAgentInput,
) -> AgentOutput:
    """
    Execute a structured task using OpenAGI's TaskerAgent with predefined steps.

    Args:
        ctx: Kernel context containing invocation information
        payload: Contains 'task' (str) and 'todos' (list of str steps)

    Returns:
        AgentOutput with success status and result message
    """
    if not payload or not payload.get("task"):
        raise ValueError("task is required")

    if not payload.get("todos") or not isinstance(payload["todos"], list):
        raise ValueError("todos must be a non-empty list of steps")

    task = payload["task"]
    todos = payload["todos"]

    async with KernelBrowserSession(record_replay=False) as session:
        print("Kernel browser live view url:", session.live_view_url)

        provider = KernelScreenshotProvider(session)
        handler = KernelActionHandler(session)

        agent = TaskerAgent(
            api_key=api_key,
            base_url=os.getenv("OAGI_BASE_URL", "https://api.agiopen.org"),
        )

        agent.set_task(task=task, todos=todos)

        print(f"\nExecuting task: {task}")
        print(f"Steps: {todos}\n")

        result = await agent.execute(
            instruction="",
            action_handler=handler,
            image_provider=provider,
        )

        return {
            "success": result,
            "result": f"TaskerAgent completed. Task: {task}. Success: {result}",
        }
