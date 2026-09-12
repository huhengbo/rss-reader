# Security Policy

## Reporting a vulnerability

Please report security-sensitive issues privately to the repository owner instead of opening a public issue when the report contains credentials, private feed URLs, webhook addresses, tokens, or exploit details.

## Secrets and runtime configuration

This repository must not contain production credentials or private RSS URLs.

- Keep `config.json` local and untracked.
- Use `config.example.json` as the committed template.
- Prefer environment variables for notification credentials in deployments.
- Rotate any credential or authenticated URL that was committed to Git history.
- Avoid posting full webhook URLs, tokens, or authenticated feed URLs in logs and issues.

Supported secret environment variables:

- `RSS_READER_FEISHU_API`
- `RSS_READER_DINGTALK_WEBHOOK`
- `RSS_READER_DINGTALK_SIGN`
- `RSS_READER_TELEGRAM_API`
- `RSS_READER_TELEGRAM_CHAT_ID`
- `RSS_READER_TELEGRAM_TOKEN`

Operational overrides:

- `RSS_READER_CONFIG`
- `RSS_READER_PORT`
- `RSS_READER_ARCHIVES`
