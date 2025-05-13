#!/usr/bin/env bun
import { Kernel } from '@onkernel/sdk';
import chalk from 'chalk';
import { Command } from 'commander';
import fs, { createReadStream } from 'fs';
import path from 'path';
import * as tmp from 'tmp';
import { getPackageVersion, zipDirectory } from './lib/util';

const program = new Command();

if (process.argv.length === 3 && ['-v', '--version'].includes(process.argv[2]!)) {
  console.log(getPackageVersion());
  process.exit(0);
}

program.name('kernel').description('CLI for Kernel deployment and invocation');

program
  .command('deploy')
  .description('Deploy a Kernel application')
  .argument('<entrypoint>', 'Path to entrypoint file (TypeScript or Python)')
  .option('--version <version>', 'Specify a version for the app (default: latest)')
  .option('--force', 'Allow overwrite of an existing version with the same name')
  .action(async (entrypoint, options) => {
    let { version: versionArg, force } = options;
    // if version not specified, use latest
    // if version and force not specified, user latest and force true
    if (!versionArg) {
      versionArg = 'latest';
      if (force && force !== 'true') {
        console.error('Error: --force must be used when version is latest');
        process.exit(1);
      } else if (!force) {
        force = 'true';
      }
    }
    const resolvedEntrypoint = path.resolve(entrypoint);
    if (!fs.existsSync(resolvedEntrypoint)) {
      console.error(`Error: Entrypoint ${resolvedEntrypoint} doesn't exist`);
      process.exit(1);
    }
    const sourceDir = path.dirname(resolvedEntrypoint); // TODO: handle nested entrypoint, i.e. ./src/entrypoint.ts

    if (!process.env['KERNEL_API_KEY']) {
      console.error('Error: KERNEL_API_KEY environment variable is not set');
      console.error('Please set your Kernel API key using: export KERNEL_API_KEY=your_api_key');
      process.exit(1);
    }
    const client = new Kernel();

    console.log(chalk.green(`Compressing files...`));
    const tmpZipFile = tmp.fileSync({ postfix: '.zip' });
    try {
      await zipDirectory(path.join(sourceDir), tmpZipFile.name);
      console.log(chalk.green(`Deploying app (version: ${versionArg})...`));
      const response = await client.apps.deploy(
        {
          file: createReadStream(tmpZipFile.name),
          version: versionArg,
          force,
          entrypointRelPath: path.relative(sourceDir, resolvedEntrypoint),
        },
        { maxRetries: 0 },
      );

      // todo: pull app name from response
      if (!response.success) {
        console.error('Error deploying to Kernel:', response.message);
        process.exit(1);
      }

      for (const app of response.apps) {
        console.log(
          chalk.green(
            `App "${app.name}" successfully deployed to Kernel with action${app.actions.length > 1 ? 's' : ''}: ${app.actions.map((a) => a.name).join(', ')}`,
          ),
        );
        console.log(
          `You can invoke it with: kernel invoke${versionArg !== 'latest' ? ` --version ${versionArg}` : ''} ${quoteIfNeeded(app.name)} ${quoteIfNeeded(app.actions[0]!.name)} '{ ... JSON payload ... }'`,
        );
      }
    } catch (error) {
      console.error('Error deploying to Kernel:', error);
      process.exit(1);
    } finally {
      // Clean up temp file
      tmpZipFile.removeCallback();
    }
  });

function quoteIfNeeded(str: string) {
  if (str.includes(' ')) {
    return `"${str}"`;
  }
  return str;
}

program
  .command('invoke')
  .description('Invoke a deployed Kernel application')
  .option('--version <version>', 'Specify a version of the app to invoke')
  .argument('<app_name>', 'Name of the application to invoke')
  .argument('<action_name>', 'Name of the action to invoke')
  .argument('<payload>', 'JSON payload to send to the application')
  .action(async (appName, actionName, payload, options) => {
    let parsedPayload;
    try {
      parsedPayload = JSON.parse(payload);
    } catch (error) {
      console.error('Error: Invalid JSON payload');
      process.exit(1);
    }

    if (!process.env['KERNEL_API_KEY']) {
      console.error('Error: KERNEL_API_KEY environment variable is not set');
      console.error('Please set your Kernel API key using: export KERNEL_API_KEY=your_api_key');
      process.exit(1);
    }
    const client = new Kernel();

    console.log(`Invoking "${appName}" with action "${actionName}" and payload:`);
    console.log(JSON.stringify(parsedPayload, null, 2));

    try {
      const response = await client.apps.invoke({
        appName,
        actionName,
        payload,
        ...(options.version && { version: options.version }),
      });

      console.log('Result:');
      console.log(JSON.stringify(JSON.parse(response.output || '{}'), null, 2));
    } catch (error) {
      console.error('Error invoking application:', error);
      process.exit(1);
    }
    return;
  });

program.parse();
