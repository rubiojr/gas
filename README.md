# GitHub Activity Summarizer for Zed

This repository contains a Zed extension called "GitHub Activity Summarizer" (GAS) that helps you retrieve and display your recent GitHub activity within the Zed editor.

## Overview

The extension connects to GitHub's API to fetch your recent activity (comments, issue interactions, and pull request participation) and presents it in a formatted summary. This allows you to quickly catch up on your GitHub activity without leaving your editor.

![screencast](/docs/screencast1.mp4)

## Components

The extension consists of two main parts:

1. **Zed Extension** (Rust code): Handles integration with Zed editor
2. **Context Server** (Go code): Communicates with GitHub API and formats the activity data

## Installation

### Prerequisites

- [Zed editor](https://zed.dev/)
- [GitHub CLI](https://cli.github.com/) installed and authenticated
- Go 1.23+ (for running the server component)
- Rust must be installed via rustup

### Setup

1. Build and install the server component:
   ```
   cd server
   go install ./cmd/github-gas-server
   ```
2. Add the compiled server binary to your PATH, if the binary is not already in your PATH.

## Configuration

The extension can be configured in your Zed project settings:

```json
{
  "gas": {
    "author": "rubiojr",                             // Required: GitHub username
    "repositories": ["owner/repo1", "owner/repo2"],  // Optional: specific repositories to track
    "query_extra": "is:open",                        // Optional: additional GitHub search filters
    "from_date": "1 week ago"                        // Optional: time range to fetch activity from
  }
}
```

If no options are provided, the extension will fetch activity from all repositories you have access to, since last week (7 days ago).

## Usage

When installed, the extension adds a "/gas" prompt to Zed. Triggering this prompt will fetch and display your recent GitHub activity in a formatted view, including:

- Issue and pull request links
- Comment content
- Creation dates
- Repository information

## Architecture

- **Rust Component**: Handles Zed extension integration, settings management, and launching the context server
- **Go Context Server**:
  - Uses GitHub API to fetch user activity
  - Formats the data for display
  - Communicates with Zed using MCP protocol
  - Supports natural language date parsing (e.g., "2 weeks ago")

## Authentication

The Go component uses GitHub authentication from:
1. A token file at `~/.config/github-gas-server/token`
2. GitHub CLI token (`gh auth token`)

In that particular order.
