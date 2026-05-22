# Discord

Use the Discord HTTP API via `one discord <action>`. Bot token via `Authorization: Bot <token>` (not Bearer).

## Channels
- `channels.read`, `channels.update`, `channels.delete`
- `channels.typing`

## Messages
- `channels.messages.list`, `channels.messages.read`
- `channels.messages.create`, `channels.messages.update`, `channels.messages.delete`
- `channels.messages.bulk_delete`

## Reactions
- `reactions.add`, `reactions.delete`, `reactions.list`

## Threads
- `threads.create`, `threads.create_from_message`
- `threads.join`, `threads.leave`

## Guilds
- `guilds.read`, `guilds.list_channels`
- `guilds.members.list`, `guilds.members.read`, `guilds.members.update`
- `guilds.members.kick`
- `guilds.bans.list`, `guilds.bans.create`, `guilds.bans.delete`
- `guilds.roles.list`, `guilds.roles.create`, `guilds.roles.update`, `guilds.roles.delete`

## Users + DMs
- `users.me`, `users.read`, `dms.create`

## Webhooks
- `webhooks.list`, `webhooks.read`, `webhooks.create`, `webhooks.update`, `webhooks.delete`, `webhooks.execute`

## Application commands
- `application_commands.list`, `application_commands.create`, `application_commands.update`, `application_commands.delete`

## Invites + audit
- `invites.read`, `invites.delete`
- `audit_log.read`

## Notes
- Message listing uses snowflake cursors: `before` / `after` / `around`, `limit` max 100.
- `guilds.members.list` and `audit_log.read` cap `limit` at 100/1000 respectively.
- 429 responses respect `X-RateLimit-Reset-After`; the runtime retries with backoff.
