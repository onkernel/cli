# QA Testing for `kernel create`

This command runs QA testing for all `kernel create` template permutations.

## Overview

You will build the CLI, create all template variations, deploy them, and provide invoke commands for manual testing.

## Step 1: Build the CLI

```bash
cd /Users/rafaelgarcia/code/onkernel/cli
make build
```

The built binary will be at `./bin/kernel`.

## Step 2: Create QA Directory

Create a timestamped QA directory:

```bash
QA_DIR="./qa-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$QA_DIR"
cd "$QA_DIR"
```

## Step 3: Get KERNEL_API_KEY

**STOP and ask the human for their `KERNEL_API_KEY`.**

This is required for all deployments. Store it for use in deploy commands:

```bash
export KERNEL_API_KEY="<value from human>"
```

## Step 4: Create All Templates

Use the built CLI binary with non-interactive flags. The command format is:

```bash
../bin/kernel create -n <name> -l <language> -t <template>
```

### Template Matrix

Here are all valid language + template combinations:

| Language    | Template         | Folder Name              | Needs Env File | Required Env Vars                    |
|-------------|------------------|--------------------------|----------------|--------------------------------------|
| typescript  | sample-app       | ts-sample-app            | No             | -                                    |
| typescript  | captcha-solver   | ts-captcha-solver        | No             | -                                    |
| typescript  | stagehand        | ts-stagehand             | Yes            | OPENAI_API_KEY                       |
| typescript  | computer-use     | ts-computer-use          | Yes            | ANTHROPIC_API_KEY                    |
| typescript  | magnitude        | ts-magnitude             | Yes            | ANTHROPIC_API_KEY                    |
| typescript  | cua              | ts-cua                   | Yes            | OPENAI_API_KEY                       |
| typescript  | gemini-cua       | ts-gemini-cua            | Yes            | GOOGLE_API_KEY, OPENAI_API_KEY       |
| python      | sample-app       | py-sample-app            | No             | -                                    |
| python      | captcha-solver   | py-captcha-solver        | No             | -                                    |
| python      | browser-use      | py-browser-use           | Yes            | OPENAI_API_KEY                       |
| python      | computer-use     | py-computer-use          | Yes            | ANTHROPIC_API_KEY                    |
| python      | cua              | py-cua                   | Yes            | OPENAI_API_KEY                       |

### Create Commands

Run each of these (they are non-interactive when all flags are provided):

```bash
# TypeScript templates
../bin/kernel create -n ts-sample-app -l typescript -t sample-app
../bin/kernel create -n ts-captcha-solver -l typescript -t captcha-solver
../bin/kernel create -n ts-stagehand -l typescript -t stagehand
../bin/kernel create -n ts-computer-use -l typescript -t computer-use
../bin/kernel create -n ts-magnitude -l typescript -t magnitude
../bin/kernel create -n ts-cua -l typescript -t cua
../bin/kernel create -n ts-gemini-cua -l typescript -t gemini-cua

# Python templates
../bin/kernel create -n py-sample-app -l python -t sample-app
../bin/kernel create -n py-captcha-solver -l python -t captcha-solver
../bin/kernel create -n py-browser-use -l python -t browser-use
../bin/kernel create -n py-computer-use -l python -t computer-use
../bin/kernel create -n py-cua -l python -t cua
```

## Step 5: Deploy Each Template

For each template directory, you need to:

1. `cd` into the directory
2. If `NeedsEnvFile` is true, **STOP and ask the human** for the required API keys
3. Create a `.env` file with those values
4. Run the deploy command

### Deploy Commands by Template

#### Templates WITHOUT env files (deploy directly):

```bash
# ts-sample-app
cd ts-sample-app && ../bin/kernel deploy index.ts && cd ..

# ts-captcha-solver
cd ts-captcha-solver && ../bin/kernel deploy index.ts && cd ..

# py-sample-app
cd py-sample-app && ../bin/kernel deploy main.py && cd ..

# py-captcha-solver
cd py-captcha-solver && ../bin/kernel deploy main.py && cd ..
```

#### Templates WITH env files (prompt human first):

For each of these, **STOP and ask the human** for the required API key(s), then create the `.env` file and deploy:

**ts-stagehand** (needs OPENAI_API_KEY):
```bash
cd ts-stagehand
echo "OPENAI_API_KEY=<value from human>" > .env
../bin/kernel deploy index.ts --env-file .env
cd ..
```

**ts-computer-use** (needs ANTHROPIC_API_KEY):
```bash
cd ts-computer-use
echo "ANTHROPIC_API_KEY=<value from human>" > .env
../bin/kernel deploy index.ts --env-file .env
cd ..
```

**ts-magnitude** (needs ANTHROPIC_API_KEY):
```bash
cd ts-magnitude
echo "ANTHROPIC_API_KEY=<value from human>" > .env
../bin/kernel deploy index.ts --env-file .env
cd ..
```

**ts-cua** (needs OPENAI_API_KEY):
```bash
cd ts-cua
echo "OPENAI_API_KEY=<value from human>" > .env
../bin/kernel deploy index.ts --env-file .env
cd ..
```

**ts-gemini-cua** (needs GOOGLE_API_KEY and OPENAI_API_KEY):
```bash
cd ts-gemini-cua
cat > .env << EOF
GOOGLE_API_KEY=<value from human>
OPENAI_API_KEY=<value from human>
EOF
../bin/kernel deploy index.ts --env-file .env
cd ..
```

**py-browser-use** (needs OPENAI_API_KEY):
```bash
cd py-browser-use
echo "OPENAI_API_KEY=<value from human>" > .env
../bin/kernel deploy main.py --env-file .env
cd ..
```

**py-computer-use** (needs ANTHROPIC_API_KEY):
```bash
cd py-computer-use
echo "ANTHROPIC_API_KEY=<value from human>" > .env
../bin/kernel deploy main.py --env-file .env
cd ..
```

**py-cua** (needs OPENAI_API_KEY):
```bash
cd py-cua
echo "OPENAI_API_KEY=<value from human>" > .env
../bin/kernel deploy main.py --env-file .env
cd ..
```

## Step 6: Provide Invoke Commands

Once all deployments are complete, present the human with these invoke commands to test manually:

```bash
# TypeScript apps
kernel invoke ts-basic get-page-title --payload '{"url": "https://www.google.com"}'
kernel invoke ts-captcha-solver test-captcha-solver
kernel invoke ts-stagehand teamsize-task --payload '{"company": "Kernel"}'
kernel invoke ts-cu cu-task --payload '{"query": "Return the first url of a search result for NYC restaurant reviews Pete Wells"}'
kernel invoke ts-magnitude mag-url-extract --payload '{"url": "https://en.wikipedia.org/wiki/Special:Random"}'
kernel invoke ts-cua cua-task --payload '{"task": "Go to https://news.ycombinator.com and get the top 5 articles"}'
kernel invoke ts-gemini-cua gemini-cua-task

# Python apps
kernel invoke python-basic get-page-title --payload '{"url": "https://www.google.com"}'
kernel invoke python-captcha-solver test-captcha-solver
kernel invoke python-bu bu-task --payload '{"task": "Compare the price of gpt-4o and DeepSeek-V3"}'
kernel invoke python-cu cu-task --payload '{"query": "Return the first url of a search result for NYC restaurant reviews Pete Wells"}'
kernel invoke python-cua cua-task --payload '{"task": "Go to https://news.ycombinator.com and get the top 5 articles"}'
```

## Summary Checklist

- [ ] Built CLI with `make build`
- [ ] Created QA directory
- [ ] Got KERNEL_API_KEY from human
- [ ] Created all 12 template variations
- [ ] Got required API keys from human (OPENAI_API_KEY, ANTHROPIC_API_KEY, GOOGLE_API_KEY)
- [ ] Deployed all 12 apps
- [ ] Provided invoke commands to human for manual testing

## Cleanup

After QA is complete, the human can remove the QA directory:

```bash
cd ..
rm -rf "$QA_DIR"
```

