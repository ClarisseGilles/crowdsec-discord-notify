# crowdsec-discord-notify

Posts CrowdSec ban alerts to Discord and deletes the ban when someone presses Unban.

CrowdSec sends one JSON object per ban to `POST /alert`. This process turns that into the Discord message and listens for the button. Anyone who can see the channel can press Unban. The button deletes that ban through the CrowdSec Local API.

`POST /alert` has no shared secret. Run it on a private network that CrowdSec can reach. Do not publish it on the internet.

## Discord

Create an application at https://discord.com/developers/applications. Open Bot, add a bot, and copy the token. Leave the Message Content Intent off. Leave the Interactions Endpoint URL empty. The process connects out to Discord.

Invite it with the OAuth2 URL Generator, scope `bot`, permissions View Channels, Send Messages, and Embed Links.

In the Discord client, turn on Developer Mode under Settings → Advanced. Right-click the alerts channel and choose Copy Channel ID. The bot has to be a member of that channel. Channel membership is the only check on Unban.

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
