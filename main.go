package main

import (
	"fmt"
	"log"
	"strings"
	"regexp"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/thoj/go-ircevent"
)

var (
	channels    []string
	server      string
	nickname    string
	tls         bool
	regexString string
	regex       *regexp.Regexp
)

func main() {
	/* Read Configuration from the config file */
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.SetDefault("tls", true)
	viper.SetDefault("regex", `^[a-z]+$`)

	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	server = viper.GetString("server")
	channels = viper.GetStringSlice("channels")
	nickname = viper.GetString("nickname")
	regexString = viper.GetString("regex")

	regex = regexp.MustCompile(regexString)

	/* Connect to the server */
	con := irc.IRC(nickname, "agab")
	con.UseTLS = viper.GetBool("tls")
	err = con.Connect(server)
	if err != nil {
		log.Panic("Failed connecting")
		return
	}

	/* When ready, join all given channels */
	con.AddCallback("001", func(e *irc.Event) {
		for _, channName := range channels {
			con.Join(channName)
		}
	})

	con.AddCallback("JOIN", func(e *irc.Event) {
		nick := e.Nick
		chann := e.Arguments[0]

		con.SendRawf("MODE %s +o %s",
				chann,
				nick)
			log.Printf("Gave OP to %s in %s", nick, chann)
	})

	/* Check if you have op in given channel */
	con.AddCallback("353", func(e *irc.Event) {
		haveSufficientRights := false
		chann := strings.Split(e.Raw, " ")[4]
		for _, prefix := range [4]string{"&", "~", "@", "%"} {
			if strings.Contains(e.Message(),
				fmt.Sprintf("%s%s", prefix, nickname)) {
				log.Printf("%s%s in the channel %s",
					prefix,
					nickname,
					chann)
				haveSufficientRights = true
			}
		}

		if !haveSufficientRights {
			con.Privmsg(chann,
				"GIB OP PLS")
		}
	})

	/* The kickbot in itself */
	con.AddCallback("PRIVMSG", func(event *irc.Event) {
		nick := event.Nick
		chann := event.Arguments[0]

		if regex.MatchString(event.Message()) {
			con.SendRawf("KICK %s %s : %s",
				chann,
				nick,
				regexString)
			log.Printf("Kicked %s from %s", nick, chann)

		}
	})

	/* Watch the config file */
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		updateConfig(con)
	})

	con.Loop()
}

func updateConfig(bot *irc.Connection) error {
	if nickname != viper.GetString("nickname") {
		nickname = viper.GetString("nickname")
		bot.Nick(nickname)
		log.Printf("Updating Nick to %s", nickname)
	}

	if regexString != viper.GetString("regex") {
		regexString = viper.GetString("regex")
		r, err := regexp.Compile(regexString)
		if err != nil {
			log.Printf("Regex didn't compile: %s", err)
		} else {
			regex = r
			log.Printf("Updated regex to : %s", regexString)
		}
	}

	return nil
}
