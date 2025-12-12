# Kernel Python Sample App - OAGI CUA

This is a Kernel application that demonstrates using OpenAGI's Lux computer-use models for browser automation.

## Overview

This template provides two agent types from the [OpenAGI SDK](https://github.com/onkernel/kernel-oagi):

### AsyncDefaultAgent
Best for high-level tasks with immediate execution. Supports two models:
- `lux-actor-1`: Fast execution (~1s/step), simple linear tasks
- `lux-thinker-1`: Complex planning, comparison tasks, handling ambiguity

### TaskerAgent
Best for structured workflows with predefined steps (todos).

## Setup

1. Get your API keys:
   - **Kernel**: [dashboard.onkernel.com](https://dashboard.onkernel.com)
   - **OpenAGI**: [developer.agiopen.org](https://developer.agiopen.org)

2. Deploy the app:
```bash
kernel login
kernel deploy main.py -e OAGI_API_KEY=your_api_key --force
```

## Usage

### AsyncDefaultAgent

Execute high-level tasks with optional model selection:

```bash
# Default model (lux-actor-1)
kernel invoke python-oagi-cua oagi-default-task \
  -p '{"instruction": "Navigate to https://agiopen.org and find the pricing page"}'

# With specific model
kernel invoke python-oagi-cua oagi-default-task \
  -p '{"instruction": "Compare prices on two websites", "model": "lux-thinker-1"}'
```

### TaskerAgent

Execute structured workflows with predefined steps:

```bash
kernel invoke python-oagi-cua oagi-tasker-task \
  -p '{"task": "Navigate to OAGI documentation", "todos": ["Go to https://agiopen.org", "Click on Documentation", "Find the API reference"]}'
```

## Model Selection Guide

| Model | Best For | Avoid When |
|-------|----------|------------|
| `lux-actor-1` | Fast execution, simple linear tasks (10-20 steps) | Complex reasoning, comparison tasks |
| `lux-thinker-1` | Complex planning, comparison tasks, handling ambiguity | Low latency needs, simple click-paths |

## Resources

- [OpenAGI Documentation](https://developer.agiopen.org)
- [Kernel Documentation](https://onkernel.com/docs/quickstart)
- [Kernel + OpenAGI Template Repository](https://github.com/onkernel/kernel-oagi)
