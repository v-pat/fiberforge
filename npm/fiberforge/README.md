# fiberforge-cli

The official NPM wrapper for [FiberForge](https://github.com/v-pat/fiberforge), a deterministic, schema-driven Go Fiber project scaffolder with an embedded MCP server.

This package simply downloads and wraps the pre-compiled Go binary, making it extremely easy to use FiberForge without needing a Go environment installed.

## Usage

You can use FiberForge directly via `npx` without installing it globally:

```bash
npx fiberforge-cli scaffold examples/blog.yaml
```

To create a new project interactively:

```bash
npx fiberforge-cli init
```

## Global Installation

```bash
npm install -g fiberforge-cli

# Now you can use it anywhere
fiberforge init
```

## Features
- Schema-driven generation of fully-featured Go Fiber applications
- Deterministic code generation (No AI Hallucinations)
- Full MCP Server embedded inside the CLI
- Generates Models, Controllers, Services, Auth, Docker, GitHub Actions, and more.

Please visit the [main repository](https://github.com/v-pat/fiberforge) for full documentation and schema examples.

