# Slack

Use the Slack Web API via `one slack <action>`. Auth: Bot Token (`xoxb-...`) as PAT.

**IMPORTANT**: All Slack endpoints return HTTP 200. Always inspect `ok` in the response body. On `ok: false`, the `error` field contains the reason (e.g. `channel_not_found`, `not_in_channel`, `invalid_auth`, `missing_scope`, `ratelimited`). The CLI runtime cannot detect these — it's the agent's job.

Channel IDs (`C…`, `D…`, `G…`) are preferred over names. Resolve via `conversations.list` / `users.lookupByEmail` if needed.

## Chat
- `chat.postMessage`, `chat.update`, `chat.delete`, `chat.postEphemeral`
- `chat.scheduleMessage`, `chat.deleteScheduledMessage`, `chat.scheduledMessages.list`
- `chat.getPermalink`

## Conversations (channels, DMs, groups)
- `conversations.list`, `conversations.info`, `conversations.history`, `conversations.replies`, `conversations.members`
- `conversations.open` (open DM), `conversations.create`, `conversations.archive`
- `conversations.join`, `conversations.leave`, `conversations.invite`, `conversations.kick`
- `conversations.setTopic`, `conversations.setPurpose`

## Users
- `users.list`, `users.info`, `users.lookupByEmail`

## Files
- `files.list`, `files.info`, `files.delete`
- Upload flow (3 steps):
  1. `files.getUploadURLExternal` → returns `upload_url` and `file_id`
  2. The agent POSTs raw file bytes to `upload_url` (not a One CLI action — do it directly with curl/HTTP)
  3. `files.completeUploadExternal` with `files=[{id, title}]` and optional `channel_id` to share

## Reactions
- `reactions.add`, `reactions.remove`, `reactions.get`

## Pins
- `pins.add`, `pins.remove`

## Bookmarks
- `bookmarks.list`, `bookmarks.add`, `bookmarks.edit`, `bookmarks.remove`

## Search
- `search.messages` (requires user token scope `search:read`; bot tokens cannot search)

## Misc
- `team.info`, `dnd.info`, `usergroups.list`

## Pagination
List endpoints use cursor pagination. Pass `cursor` from `response_metadata.next_cursor` to fetch the next page. The runtime auto-pages up to `max_pages`.

## Rich content
For `blocks` and `attachments`, pass a JSON array. Slack renders Block Kit blocks; prefer them over markdown for structured messages.
