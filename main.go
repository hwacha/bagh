package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	CommandLine   bool
	Secret        bool
	ApplicationID string
	token         string
	Games         = make(map[string]SessionState)
)

func init() {
	flag.BoolVar(&CommandLine, "c", false, "Play on command line")
	flag.BoolVar(&Secret, "s", false, "Make command line action inputs secret")
	flag.Parse()
}

func main() {
	if CommandLine {
		runGameCommandLine()
		return
	}

	godotenv.Load()
	token = os.Getenv("BOT_TOKEN")

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session: ", err)
		return
	}

	dg.AddHandler(handleReady)
	dg.AddHandler(handleGuildCreate)
	dg.AddHandler(handleGuildMemberRemove)
	dg.AddHandler(handleGuildLeave)
	dg.AddHandler(handleApplicationCommand)

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsGuildMembers

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}
