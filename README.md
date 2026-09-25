# crowdsec-discord-notify

Posts CrowdSec ban alerts to Discord and deletes the ban when someone presses Unban.

CrowdSec sends one JSON object per ban to `POST /alert`. This process turns that into the Discord message and listens for the button. Anyone who can see the channel can press Unban. The button deletes that ban through the CrowdSec Local API.

`POST /alert` has no shared secret. Run it on a private network that CrowdSec can reach. Do not publish it on the internet.

## Discord

This bot deletes bans on your CrowdSec. Keep it private. With Public Bot off, only you can add it to a server. Put the alerts in a channel that only people who may unban can read. Anyone who can see that channel can press Unban.

1. Create an application at https://discord.com/developers/applications.
2. Open Bot. Add a bot if the page asks for one. Press Reset Token and copy the token. That value is `DISCORD_BOT_TOKEN`. Discord shows it once. If it leaks, reset it again and update the container.
3. Open Installation and set Install Link to None. Discord refuses to make the bot private while an install link is set. Then open Bot and turn Public Bot off.
4. On the Bot page, leave Requires OAuth2 Code Grant off. Leave Message Content Intent off, along with the other privileged intents. Leave the Interactions Endpoint URL empty. This process opens its own connection to Discord. It does not take interaction callbacks on a public URL.
5. Open OAuth2 → URL Generator. Select the `bot` scope. Under bot permissions, select View Channels, Send Messages, and Embed Links. Those three are permission integer `19456`. The URL is `https://discord.com/oauth2/authorize?client_id=APPLICATION_ID&scope=bot&permissions=19456`, with your application ID in place of `APPLICATION_ID`. Open it and add the bot to your server. You need Manage Server on that Discord server to install it.
6. In the Discord client, open User Settings → Advanced and turn on Developer Mode. Right-click the alerts channel and choose Copy Channel ID. That value is `DISCORD_CHANNEL_ID`.
7. In the channel permissions, allow the bot to view the channel, send messages, and embed links. Deny everyone else if they should not be able to unban.

CrowdSec needs a bouncer key that this process can use to delete decisions. On the official image, set `BOUNCER_KEY_DISCORD` to that key and it registers a bouncer on startup. Use the same value for `BOUNCER_KEY` here.

## Configuration

| Variable | Required | Purpose |
| --- | --- | --- |
| `DISCORD_BOT_TOKEN` | yes | Bot token from the Discord developer portal |
| `DISCORD_CHANNEL_ID` | yes | Channel that receives alerts |
| `BOUNCER_KEY` | yes | CrowdSec bouncer key sent as `X-Api-Key` when deleting a decision |
| `LAPI_URL` | yes | CrowdSec Local API, for example `http://crowdsec:8080` |
| `GEOAPIFY_API_KEY` | no | Draws a map on the alert. If this is unset, the message is sent without a map |

## Run

Listens on port 8080.

```bash
podman run --rm -p 8080:8080 \
  -e DISCORD_BOT_TOKEN \
  -e DISCORD_CHANNEL_ID \
  -e BOUNCER_KEY \
  -e LAPI_URL \
  -e GEOAPIFY_API_KEY \
  ghcr.io/clarissegilles/crowdsec-discord-notify:0.1.0
```

`docker run` takes the same arguments. The image is `linux/amd64` and `linux/arm64`.

The image does not contain your Discord token, bouncer key, or CrowdSec address. Those exist only in the environment of the container you run. Leave the GitHub package private.

A CrowdSec CTI API key is configured on CrowdSec, not here. The notification below uses it to fill `city` and `maliciousness`.

## CrowdSec notification

Save this as a CrowdSec HTTP notification. The profile that bans must list this notification by name (`discord` below). Only decisions of type `ban` are sent.

`url` is this process's `POST /alert`. Change the host if the container name is different. CrowdSec needs its CTI API key configured, or `city` and `maliciousness` stay empty.

```yaml
type: http
name: discord
log_level: info
format: |
  {{- $out := list -}}
  {{- range . -}}
    {{- $alert := . -}}
    {{- range .Decisions -}}
      {{- if eq .Type "ban" -}}
        {{- $city := "" -}}
        {{- $country := "" -}}
        {{- $mal := 0.0 -}}
        {{- if $alert.Source.Cn -}}{{- $country = printf "%s" $alert.Source.Cn -}}{{- end -}}
        {{- if eq .Scope "Ip" -}}
          {{- $cti := .Value | CrowdsecCTI -}}
          {{- if $cti.Location.City -}}
            {{- $city = printf "%s" $cti.Location.City -}}
            {{- $mal = mulf $cti.GetMaliciousnessScore 100 | floor -}}
            {{- if gt $mal 100.0 -}}{{- $mal = 100.0 -}}{{- end -}}
          {{- end -}}
        {{- end -}}
        {{- $out = append $out (dict
              "scope" .Scope
              "value" .Value
              "scenario" .Scenario
              "duration" .Duration
              "country" $country
              "latitude" $alert.Source.Latitude
              "longitude" $alert.Source.Longitude
              "city" $city
              "maliciousness" $mal
              "domain" (GetMeta $alert "target_fqdn" | uniq | join "\n")
              "meta" $alert.Meta) -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
  {{- $out | toJson -}}
url: http://crowdsec-discord-notify:8080/alert
method: POST
headers:
  Content-Type: application/json
```

## License

[MIT](LICENSE)
