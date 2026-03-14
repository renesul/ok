---
name: browser
description: "Control a real web browser: navigate pages, click elements, fill forms, extract content, take screenshots. Requires playwright-mcp MCP server."
metadata: {"ok":{"emoji":"🌐"}}
---

# Browser Automation Skill

This skill uses the **playwright-mcp** MCP server to control a real browser (Chromium). Full JavaScript rendering, form submission, login flows, scraping, and more.

## Setup (one-time)

1. Install Node.js and the playwright-mcp server:
```bash
npm install -g @playwright/mcp
```

2. Add to `config.json`:
```json
{
  "mcp_servers": [
    {
      "name": "browser",
      "enabled": true,
      "transport": "stdio",
      "command": "npx",
      "args": ["@playwright/mcp", "--headless"],
      "tool_prefix": "browser_"
    }
  ]
}
```

3. Restart OK. The browser tools will appear automatically.

## Available tools (after setup)

- `browser_navigate` — Navigate to a URL
- `browser_screenshot` — Take a screenshot
- `browser_click` — Click an element (by CSS selector or text)
- `browser_type` — Type text into an input field
- `browser_evaluate` — Run JavaScript and return result
- `browser_get_text` — Extract all text from the current page
- `browser_wait_for` — Wait for an element to appear

## Common workflows

### Extract content from a page
```
1. browser_navigate url="https://example.com"
2. browser_get_text → returns page text
```

### Log in and do something
```
1. browser_navigate url="https://app.example.com/login"
2. browser_type selector="#email" text="user@example.com"
3. browser_type selector="#password" text="secret"
4. browser_click selector="button[type=submit]"
5. browser_wait_for selector=".dashboard"
6. browser_screenshot → confirm logged in
```

### Fill a form
```
1. browser_navigate url="https://site.com/form"
2. browser_type selector="input[name=name]" text="John"
3. browser_type selector="input[name=email]" text="john@example.com"
4. browser_click selector="button.submit"
```

## Tips
- Use `browser_screenshot` to debug what the page looks like
- Prefer `text=Button Label` selectors when CSS selectors are complex
- For dynamic pages, use `browser_wait_for` before interacting
