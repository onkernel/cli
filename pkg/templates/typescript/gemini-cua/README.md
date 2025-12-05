# Kernel TypeScript SDK + Stagehand + Gemini Computer Use Agent

A Kernel application that demonstrates Computer Use Agent (CUA) capabilities using Google's Gemini 2.5 model with Stagehand for browser automation.

## What It Does

This app uses [Gemini 2.5's computer use model](https://blog.google/technology/google-deepmind/gemini-computer-use-model/) capabilities to autonomously navigate websites and complete tasks. The example task searches for Kernel's company page on YCombinator and writes a blog post about their product.

## Setup

1. **Add your API keys as environment variables:**
   - `KERNEL_API_KEY` - Get from [Kernel dashboard](https://dashboard.onkernel.com/sign-in)
   - `GOOGLE_API_KEY` - Get from [Google AI Studio](https://aistudio.google.com/apikey)
   - `OPENAI_API_KEY` - Get from [OpenAI platform](https://platform.openai.com/api-keys)

## Running Locally

Execute the script directly with tsx:

```bash
npx tsx index.ts
```

This runs the agent without a Kernel invocation context and provides the browser live view URL for debugging.

## Deploying to Kernel

1. **Deploy the application:**
   ```bash
   kernel deploy index.ts --env GOOGLE_API_KEY=XXX --env OPENAI_API_KEY=XXX
   ```

2. **Invoke the action:**
   ```bash
   kernel invoke ts-gemini-cua gemini-cua-task
   ```

The action creates a Kernel-managed browser and associates it with the invocation for tracking and monitoring.

## Documentation

- [Kernel Documentation](https://docs.onkernel.com/quickstart)
- [Kernel Stagehand Guide](https://www.onkernel.com/docs/integrations/stagehand)
- [Gemini 2.5 Computer Use](https://blog.google/technology/google-deepmind/gemini-computer-use-model/)
