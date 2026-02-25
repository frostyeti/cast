---
title: "8-create-docs"
type: chore
tags: ["chore"]
---

# Create Docs

## Implementation Plan

1. **Scaffold Astro Starlight (`docs/`):**
   - Temporarily backup any existing project planning files located in `docs/` (such as `docs/issues/`).
   - Remove the existing `docs/` directory.
   - Run `bun create astro@latest docs --template starlight` to initialize a new Starlight documentation site in the `docs/` folder (using `bun` instead of `npm`).
   - Migrate the backed-up `issues/` files back into the site (e.g., under `docs/src/content/docs/issues/` or a separate tracking folder) if desired.

2. **Configure Starlight (`docs/astro.config.mjs`):**
   - Enable Starlight's built-in `Pagefind` for local, fast search capabilities.
   - Install and configure the `@astrojs/rss` integration using `bun add` to generate an RSS feed for announcements or blog posts.
   - Organize the sidebar and consider enabling versioning to support different releases of `cast`.

3. **Develop Content (`docs/src/content/docs/`):**
   - **Core Concepts:** Getting started, Castfile structure, Workspaces.
   - **CLI Reference:** Document commands such as `cast <task>`, `cast tool install deno/mise`, `cast update`, and `cast tools docker purge`.
   - **Use Cases & Examples:** Provide comprehensive YAML examples for:
     - Managing secrets securely.
     - Utilizing Remote Tasks (Git/JSR) and Remote Modules.
     - Defining Docker Tasks and Deno Tasks.
     - Practical guides for Build/Deploy workflows and ETL tasks.

4. **CI/CD Deployment (Cloudflare Pages):**
   - Create a `.github/workflows/docs.yml` workflow for automated deployment using a `bun` environment setup step.
   - Alternatively, configure Cloudflare Pages via the dashboard to directly track the GitHub repository with the framework preset set to Astro/Bun.
   - The build command will be `bun run build` (executed within the `docs/` directory) and the output artifact directory will be `docs/dist/`.