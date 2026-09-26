package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

func (a *App) postBan(item alert) error {
	_, err := a.discord.ChannelMessageSendComplex(a.config.DiscordChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{a.embed(item)},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Unban",
					Style:    discordgo.DangerButton,
					CustomID: "unban:" + item.Scope + ":" + item.Value,
				},
			}},
		},
	})
	return err
}

func (a *App) onClick(session *discordgo.Session, click *discordgo.InteractionCreate) {
	if click.Type != discordgo.InteractionMessageComponent {
		return
	}
	parts := strings.SplitN(click.MessageComponentData().CustomID, ":", 3)
	if len(parts) != 3 || parts[0] != "unban" || parts[1] == "" || parts[2] == "" {
		return
	}
	scope, value := parts[1], parts[2]
	err := session.InteractionRespond(click.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil {
		log.Printf("ack unban %s: %v", value, err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	text := "Unbanned " + value
	if err = a.deleteDecision(ctx, scope, value); err != nil {
		log.Printf("delete decision %s %s: %v", scope, value, err)
		text = "CrowdSec did not remove " + value
	} else {
		log.Printf("unbanned %s %s by %s", scope, value, clicker(click))
	}
	if _, err = session.InteractionResponseEdit(click.Interaction, &discordgo.WebhookEdit{Content: &text}); err != nil {
		log.Printf("edit unban reply: %v", err)
	}
}

func clicker(click *discordgo.InteractionCreate) string {
	if click.Member != nil && click.Member.User != nil {
		return click.Member.User.Username
	}
	if click.User != nil {
		return click.User.Username
	}
	return "unknown"
}

func (a *App) embed(item alert) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "Crowdsec Alert",
		Color:       16711680,
		URL:         "https://app.crowdsec.net/cti/" + item.Value,
		Description: "Potential threat detected. View details in [Crowdsec Console](<https://app.crowdsec.net/cti/" + item.Value + ">)",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Scenario", Value: "`" + item.Scenario + "`", Inline: true},
			{Name: "IP", Value: "[" + item.Value + "](<https://www.shodan.io/host/" + item.Value + ">)", Inline: true},
			{Name: "Ban Duration", Value: item.Duration, Inline: true},
		},
	}
	if item.Domain != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Domain", Value: "`" + strings.ReplaceAll(item.Domain, "\n", "`\n`") + "`", Inline: true,
		})
	}
	if item.Country != "" {
		if a.config.GeoapifyAPIKey != "" {
			embed.Image = &discordgo.MessageEmbedImage{URL: mapImage(item, a.config.GeoapifyAPIKey)}
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Country", Value: item.Country + " :flag_" + strings.ToLower(item.Country) + ":", Inline: true,
		})
		if item.City != "" {
			embed.Fields = append(embed.Fields,
				&discordgo.MessageEmbedField{Name: "City", Value: item.City, Inline: true},
				&discordgo.MessageEmbedField{Name: "Maliciousness", Value: fmt.Sprintf("%.0f %%", item.Maliciousness), Inline: true},
			)
		}
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Location", Value: "Unknown :pirate_flag:"})
	}
	n := 0
	for _, meta := range item.Meta {
		if n == 20 {
			break
		}
		value := clip(metaText(meta.Value))
		if meta.Key == "" || value == "" {
			continue
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: meta.Key, Value: value})
		n++
	}
	return embed
}

func mapImage(item alert, key string) string {
	point := coord(item.Longitude) + "," + coord(item.Latitude)
	return "https://maps.geoapify.com/v1/staticmap?style=osm-bright-grey&width=600&height=400&center=lonlat:" + point +
		"&zoom=8.1848&marker=lonlat:" + point + ";type:awesome;color:%23655e90;size:large;icon:industry|lonlat:" + point +
		";type:material;color:%23ff3421;icontype:awesome&scaleFactor=2&apiKey=" + key
}

func metaText(raw string) string {
	raw = strings.NewReplacer(`"`, "`", "[", "", "]", "").Replace(raw)
	return strings.Join(strings.Split(raw, ","), "\n")
}

func coord(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func clip(s string) string {
	if utf8.RuneCountInString(s) <= 1000 {
		return s
	}
	return string([]rune(s)[:1000])
}
