package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const APPLICATION_ID = "1291027616702402632"

var (
	CommandLine bool
	Secret      bool
	Token       string
	Games       = make(map[string]SessionState)
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.BoolVar(&CommandLine, "c", false, "Play on command line")
	flag.BoolVar(&Secret, "s", false, "Make command line action inputs secret")
	flag.Parse()
}

func main() {
	if CommandLine {
		runGameCommandLine()
		return
	}

	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session: ", err)
		return
	}

	dg.AddHandler(handleReady)
	dg.AddHandler(handleGuildCreate)
	dg.AddHandler(handleGuildMemberRemove)
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
