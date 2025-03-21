# GitHub Activity Summarizer for Zed

This repository contains a Zed extension called "GitHub Activity Summarizer" (GAS) that helps you retrieve and display your recent GitHub activity within the Zed editor.

## Overview

The extension is a [Zed context server](https://zed.dev/docs/extensions/context-servers) that connects to GitHub's API to fetch your recent activity (comments, issue interactions, and pull request participation) and presents it in a formatted summary. This allows you to quickly catch up on your GitHub activity without leaving your editor.

https://github.com/user-attachments/assets/0856b1a8-8dbc-4453-ae51-ef40bf25fbca

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

## Setup

### Installation

#### From the marketplace

Install it from the Zed.

#### From source (dev extension)

Make sure Go and Rust (from rustup) are installed before proceeding.

1. Clone the repository: `git clone https://github.com/rubiojr/gas`
2. Install the extension from `Extensions -> Install Dev Extension`.

### Configuration

The extension can be configured in your Zed project settings:

```json
"context_servers": {
  "gas": {
    "settings": {
      "author": "",                                       // Optional: GitHub username, defaults to your username
      "repositories": [],                                 // Optional: specific repositories to include (defaults to all)
      "query_extra": "",                                  // Optional: additional GitHub search query filters (defaults to none)
      "from_date": "1 week ago",                           // Optional: time range to fetch activity from (defaults to 7 days ago)
      "auth_type": "all"                                  // Optional: authentication type (defaults to "all")
    }
  }
}
```

By default, the extension will use the GitHub CLI (`gh`) for authentication. If that's undesired, see the Authentication section for alternatives.

Valid authentication types are:

- `all`: tries all authentication methods described in the Authentication section.
- `cli`: Use GitHub CLI (`gh`) for authentication.
- `file`: Read the token from `~/.config/github-gas-server/token`
- `keyring`: Read the token from the system keyring. (macOS and Linux only)

If no options are provided, the extension will fetch activity from all repositories you have access to, since last week (7 days ago).

`repositories` is a list of `owner/repo` to include in the search query.

The `github-gas-server` binary is downloaded automatically from the GitHub repository. If you want to specify a custom path, you can do so by setting the `path` option in the `command` section:

```json
"context_servers": {
  "gas": {
    "command": { // path to the server binary is optional, it'll be downloaded automatically
      "path": "/path/to/github-gas-server",
      "args": ["stdio"]
    },
    "settings": {
      "author": "rubiojr",                             // Required: GitHub username
      "repositories": ["owner/repo1", "owner/repo2"],  // Optional: specific repositories to include (defaults to all)
      "query_extra": "-org:github",                    // Optional: additional GitHub search query filters (defaults to none)
      "from_date": "1 week ago"                        // Optional: time range to fetch activity from (defaults to 7 days ago)
    }
  }
}
```

#### query_extra tips

- Use `-type:issue` or `-type:pr` to exclude issues or pull requests from the search results.
- Use `-org:github` to exclude prs and issues from the given org (github in this case).

Use any search filter documented at https://docs.github.com/en/search-github/searching-on-github/searching-issues-and-pull-requests

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

1. OS Keychain (macOS) or GNOME Keyring (Linux)
2. A token file at `~/.config/github-gas-server/token` (simply drop the token then, no specific format required)
3. GitHub CLI token (`gh auth token`)

In that particular order.

### Adding the token to the GNOME Keyring

From the command line:

```
secret-tool store --label="Token for GitHub Activity Summarizer" service github-activity-summarizer
```

Then paste the token into the prompt.

### Adding the token to the macOS Keychain

```
security add-generic-password -a github-activity-summarizer -s github-activity-summarizer -w
```

Then paste the token into the prompt.
