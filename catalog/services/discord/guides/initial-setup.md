---
title: Discord — Initial Setup
---

One CLI talks to the Discord API at `https://discord.com/api/v10` using a token
(`pat` provider).

## 1. Create an application and bot token

1. Go to https://discord.com/developers/applications → **New Application**.
2. Open the **Bot** tab → **Reset Token** → copy the bot token.
3. Under **Bot → Privileged Gateway Intents**, enable only what you need
   (e.g. Server Members / Message Content) — many read actions require them.
4. Invite the bot to your server via **OAuth2 → URL Generator** (scope `bot`
   plus the permissions the actions you use need), then open the generated URL.

## 2. Store the token

```bash
one login discord           # --provider pat is the default
one capabilities discord    # confirm actions are visible
```

## Notes

- Bot tokens are sent as `Authorization: Bot <token>`. TODO: verify — confirm
  this catalog's `pat` injection produces the `Bot ` prefix; a bare token will
  be rejected with `401`.
- The bot only sees guilds it has been invited to and channels its role can view.
