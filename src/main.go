package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	DiscordBotToken  string
	DiscordChannelID string
	LAPIURL          string
	MachineID        string
	MachinePassword  string
	GeoapifyAPIKey   string
}

type App struct {
	config  Config
	discord *discordgo.Session
	client  *http.Client
}

func main() {
	cfg := loadConfig()
	session, err := discordgo.New("Bot " + cfg.DiscordBotToken)
	if err != nil {
		log.Fatal(err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds
	app := &App{
		config:  cfg,
		discord: session,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
	session.AddHandler(app.onClick)
	if err = session.Open(); err != nil {
		log.Fatal(err)
	}
	maps := "off"
	if cfg.GeoapifyAPIKey != "" {
		maps = "on"
	}
	log.Printf("starting on :8080, lapi %s, machine %s, channel %s, maps %s", cfg.LAPIURL, cfg.MachineID, cfg.DiscordChannelID, maps)
	http.HandleFunc("POST /alert", app.receive)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadConfig() Config {
	return Config{
		DiscordBotToken:  getenv("DISCORD_BOT_TOKEN"),
		DiscordChannelID: getenv("DISCORD_CHANNEL_ID"),
		LAPIURL:          getenv("LAPI_URL"),
		MachineID:        getenv("MACHINE_ID"),
		MachinePassword:  getenv("MACHINE_PASSWORD"),
		GeoapifyAPIKey:   os.Getenv("GEOAPIFY_API_KEY"),
	}
}

func getenv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}

func (a *App) receive(w http.ResponseWriter, r *http.Request) {
	var alerts []alert
	if json.NewDecoder(r.Body).Decode(&alerts) != nil {
		log.Printf("alert: bad request")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	log.Printf("received %d ban(s)", len(alerts))
	for _, item := range alerts {
		if err := a.postBan(item); err != nil {
			log.Printf("alert %s %s: %v", item.Scope, item.Value, err)
			continue
		}
		log.Printf("posted %s %s %s", item.Scope, item.Value, item.Scenario)
	}
	w.WriteHeader(http.StatusNoContent)
}