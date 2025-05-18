# Kernel CLI

A command-line tool for deploying and invoking Kernel applications.

## Installation

```bash
brew install onkernel/tap/kernel
```

## Development Prerequisites

Install the following tools:

- Go 1.22+ ( https://go.dev/doc/install )
- [Goreleaser](https://goreleaser.com/install/)
- [chglog](https://github.com/goreleaser/chglog)

Compile the CLI:

```bash
make build   # compiles the binary to ./bin/kernel
```

Run the CLI:

```bash
./bin/kernel --help
```

## Development workflow

Useful make targets:

- `make build` – compile the project to `./bin/kernel`
- `make test` – execute unit tests
- `make lint` – run the linter (requires `golangci-lint`)
- `make changelog` – generate/update the `CHANGELOG.md` file using **chglog**
- `make release` – create a release using **goreleaser** (builds archives, homebrew formula, etc. See below)

### Releasing a new version

Prerequisites:

- Make sure you have `goreleaser` _pro_ installed via `brew install goreleaser/tap/goreleaser-pro`. You will need a license key (in 1pw), and then `export GORELEASER_KEY=<the key>`.

- Grab the NPM token for our org (in 1pw) and run `npm config set '//registry.npmjs.org/:_authToken'=<the token>`

- export a `GITHUB_TOKEN` with repo and write:packages permissions: https://github.com/settings/tokens/new?scopes=repo,write:packages.

- Make sure you are logged in to the prod AWS account with `aws sso login --sso-session=kernel` + `export AWS_PROFILE=kernel-prod`. This is necessary to publish releases to S3.

On main, run:

```bash
make release-dry-run
```

This will check that everything is working, but not actually release anything.
You should see one error about there not being a git tag, and that's fine.

To actually release, run:

```bash
export VERSION=0.1.1
git tag -a cli/v$VERSION -m "Bugfixes"
git push origin cli/v$VERSION
make release
```

The NPM publish needs some extra steps (hopefully one day goreleaser will support this, but right now it assumes download links are public github releases, which we don't have at the moment):

````bash
go run scripts/npmpublish/npmpublish.go ./.npmdisttmpl ./dist
cd .npmdist && npm publish --access public
```)

### Environment variables

The CLI requires a Kernel API key to interact with the platform:

```bash
export KERNEL_API_KEY=your_api_key
````

## Directory structure

```
packages/cli
├── cmd/          # cobra commands (root, deploy, invoke, …)
│   └── kernel/
│       └── main.go
├── pkg/          # reusable helpers (zip util, etc.)
├── .goreleaser.yaml
├── Makefile
└── README.md
```
