---
worth: maybe
where: app/bot/openai/openai.go:autoReply
added: 2026-09-22
---
# unsolicited replies cannot use context older than the snapshot window

The jev-gated reply sees only the current chat's last 10 messages from the past 30 minutes. Some good
requests depend on older discussion ("помню как-то давно бобук рекомендовал пост про выгорание, не
подскажете?"), which the reporter archive could answer. The idea: a small tool harness (recent
history, archive search, reply chain) behind the same gate. Postponed until the gated version has run
in production and been observed.

Notes from the design review, to avoid re-deriving them:

- Measure retrieval need before the gate filters anything. The gate's `answerable` question rejects
  archive-dependent requests by design, so counting only accepted cases that miss context undercounts
  the need. Label socially eligible requests for the evidence they need (recent context, archive, private
  or media context) and count across those.
- Try archive search offline first: search the reporter logs for eligible archive-needing cases, feed a
  bounded set of matches (with neighbours, author, date) to one completion, and compare against the
  recent-context answer. A store and a tool loop come only if that shows value; start with bounded
  retrieval plus one completion, add a loop only if query reformulation measurably helps.
- A multi-call answer is too slow for the synchronous `MultiBot` path. `TelegramListener.Submit` already
  sends later results (the rtjc summary workers use it), but it targets the group only, drops `ReplyTo`,
  returns on enqueue rather than delivery, and bypasses the incoming activity checks. A delayed reply
  needs a narrow envelope: reply target, expiry, outcome.
- The reporter logs keep the parent's author and text but not its message ID, so exact multi-hop reply
  chains cannot be rebuilt from them; the raw Telegram JSON in the debug log has
  `reply_to_message.message_id`.
- Retrieved old messages need attribution: an old recommendation is evidence someone said it, not that
  it is still correct.
