---
name: email
description: "Read and send emails via IMAP/SMTP. Use to check inbox, search messages, and send emails."
metadata: {"ok":{"emoji":"📧"}}
---

# Email Skill

Use the `email` tool to interact with your email inbox.

## Configuration required (config.json)

```json
{
  "integrations": {
    "email": {
      "enabled": true,
      "imap_host": "imap.gmail.com",
      "imap_port": 993,
      "imap_tls": true,
      "smtp_host": "smtp.gmail.com",
      "smtp_port": 587,
      "username": "you@gmail.com",
      "password": "your-app-password",
      "from_name": "Your Name",
      "max_fetch": 10
    }
  }
}
```

For Gmail, use an **App Password** (not your regular password): Google Account → Security → 2-Step Verification → App Passwords.

## Read recent emails

```json
{"action": "read", "count": 10}
```

## Search emails

```json
{"action": "search", "query": "FROM boss@company.com"}
{"action": "search", "query": "SUBJECT invoice UNSEEN"}
{"action": "search", "query": "UNSEEN"}
```

IMAP search criteria: `FROM`, `TO`, `SUBJECT`, `UNSEEN`, `SEEN`, `SINCE 01-Jan-2024`, `BEFORE 31-Dec-2024`

## Send email

```json
{
  "action": "send",
  "to": "recipient@example.com",
  "subject": "Hello",
  "body": "Message body here."
}
```

Multiple recipients: `"to": "a@x.com, b@y.com"`
