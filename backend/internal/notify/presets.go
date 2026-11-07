package notify

// Header is one extra HTTP header sent with a webhook.
type Header struct {
	Name  string `json:"name" maxLength:"100"`
	Value string `json:"value" maxLength:"2000"`
}

// Preset fills a webhook form for a known service. Placeholders in URLExample (UPPER_CASE) are for the admin to replace.
type Preset struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	URLExample string   `json:"urlExample"`
	Template   string   `json:"template" doc:"Liquid body; empty = the raw JSON envelope"`
	Headers    []Header `json:"headers"`
	AuthType   string   `json:"authType" enum:"none,bearer,basic"`
	Notes      string   `json:"notes"`
}

// Presets are the ready-made templates offered in the web UI (and documented in docs/notifications.md).
var Presets = []Preset{
	{ID: "raw", Name: "Raw JSON", URLExample: "https://example.org/hook", Headers: []Header{}, AuthType: "none",
		Notes: "The Tipsarr envelope {event, timestamp, dryRun, data}."},
	{ID: "discord", Name: "Discord", URLExample: "https://discord.com/api/webhooks/ID/TOKEN", Headers: []Header{}, AuthType: "none",
		Notes: "Channel webhook. With per-recipient delivery, the person is mentioned when they saved their Discord user id.",
		Template: `{%- capture mention -%}{% if notifyUser.discordId %}<@{{ notifyUser.discordId }}>{% endif %}{%- endcapture -%}
{
  "username": "Tipsarr",
  "content": {{ mention | json }},
  "embeds": [{
    "title": {{ subject | json }},
    "description": {{ message | json }},
    {% if url != "" %}"url": {{ url | json }},{% endif %}
    {% if image != "" %}"thumbnail": {"url": {{ image | json }}},{% endif %}
    "color": 6514417,
    "footer": {"text": {{ event | json }}},
    "timestamp": {{ timestamp | json }}
  }]
}`},
	{ID: "ntfy", Name: "ntfy", URLExample: "https://ntfy.sh", Headers: []Header{}, AuthType: "bearer",
		Notes: "JSON publishing to the server root; change the topic. Bearer = an ntfy access token (optional).",
		Template: `{
  "topic": "tipsarr",
  "title": {{ subject | json }},
  "message": {{ message | json }},
  {% if url != "" %}"click": {{ url | json }},{% endif %}
  {% if image != "" %}"attach": {{ image | json }},{% endif %}
  "tags": ["clapper"]
}`},
	{ID: "gotify", Name: "Gotify", URLExample: "https://gotify.example.org/message", AuthType: "none",
		Headers: []Header{{Name: "X-Gotify-Key", Value: "APP_TOKEN"}},
		Notes:   "Replace APP_TOKEN with an application token.",
		Template: `{
  "title": {{ subject | json }},
  "message": {{ message | json }},
  "priority": 5,
  "extras": {
    "client::display": {"contentType": "text/markdown"}{% if url != "" %},
    "client::notification": {"click": {"url": {{ url | json }}}{% if image != "" %}, "bigImageUrl": {{ image | json }}{% endif %}}{% endif %}
  }
}`},
	{ID: "telegram", Name: "Telegram", URLExample: "https://api.telegram.org/botBOT_TOKEN/sendMessage", Headers: []Header{}, AuthType: "none",
		Notes: "Replace BOT_TOKEN and CHAT_ID. With per-recipient delivery, each person's Telegram chat id is used when saved.",
		Template: `{
  "chat_id": {{ notifyUser.telegramChatId | default: "CHAT_ID" | json }},
  "text": {{ subject | append: nl | append: message | append: nl | append: url | json }},
  "disable_web_page_preview": false
}`},
	{ID: "slack", Name: "Slack", URLExample: "https://hooks.slack.com/services/T000/B000/XXXX", Headers: []Header{}, AuthType: "none",
		Notes: "Incoming webhook.",
		Template: `{
  "text": {{ subject | json }},
  "blocks": [{
    "type": "section",
    "text": {"type": "mrkdwn", "text": {{ "*" | append: subject | append: "*" | append: nl | append: message | json }}}{% if image != "" %},
    "accessory": {"type": "image", "image_url": {{ image | json }}, "alt_text": {{ media.title | default: subject | json }}}{% endif %}
  }]
}`},
	{ID: "pushover", Name: "Pushover", URLExample: "https://api.pushover.net/1/messages.json", Headers: []Header{}, AuthType: "none",
		Notes: "Replace APP_TOKEN and USER_KEY.",
		Template: `{
  "token": "APP_TOKEN",
  "user": "USER_KEY",
  "title": {{ subject | json }},
  "message": {{ message | json }}{% if url != "" %},
  "url": {{ url | json }},
  "url_title": "Tipsarr"{% endif %}
}`},
	{ID: "apprise", Name: "Apprise", URLExample: "http://apprise:8000/notify/KEY", Headers: []Header{}, AuthType: "none",
		Notes: "Apprise API stateful endpoint (/notify/KEY) or stateless with a \"urls\" field.",
		Template: `{
  "title": {{ subject | json }},
  "body": {{ message | append: nl | append: url | json }},
  "type": "info",
  "format": "text"
}`},
	{ID: "homeassistant", Name: "Home Assistant", URLExample: "http://homeassistant:8123/api/webhook/WEBHOOK_ID", Headers: []Header{}, AuthType: "none",
		Notes: "Webhook trigger; everything is in trigger.json.",
		Template: `{
  "event": {{ event | json }},
  "subject": {{ subject | json }},
  "message": {{ message | json }},
  "image": {{ image | json }},
  "url": {{ url | json }},
  "media": {{ media | json }},
  "requestedBy": {{ requestedBy | json }},
  "notifyUser": {{ notifyUser | json }}
}`},
}
