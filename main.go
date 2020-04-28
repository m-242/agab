package main

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/thoj/go-ircevent"
)

var (
	channels    []string
	probability int
	server      string
	nickname    string
	reasons     []string
	opRequests  []string
	tls         bool
)

func main() {
	/* Read Configuration from the config file */
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.SetDefault("reasons", [1]string{"You talk to much"})
	viper.SetDefault("opRequests", [1]string{"Can I haz OP ?"})
	viper.SetDefault("tls", true)

	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	server = viper.GetString("server")
	channels = viper.GetStringSlice("channels")
	nickname = viper.GetString("nickname")
	probability = viper.GetInt("probability")
	reasons = viper.GetStringSlice("reasons")
	opRequests = viper.GetStringSlice("opRequests")

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
				randSliceValue(opRequests))
		}
	})

	/* The kickbot in itself */
	con.AddCallback("PRIVMSG", func(event *irc.Event) {
		nick := event.Nick
		chann := event.Arguments[0]

		if rand.Intn(probability) == 1 {
			con.SendRawf("NAMES %s", chann) // Trigger permission test
			con.SendRawf("KICK %s %s : %s",
				chann,
				nick,
				randSliceValue(reasons))
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

func randSliceValue(sl []string) string {
	switch len(sl) {
	case 0:
		return "yup"
	case 1:
		return sl[0]
	default:
		return sl[rand.Intn(len(sl))]
	}
}

func updateConfig(bot *irc.Connection) error {
	if !reflect.DeepEqual(viper.GetStringSlice("channels"), channels) {
		// We part channels that aren't in the new config, and join
		// those that were added
		var (
			part []string
			join []string
		)

		newchans := make(map[string]struct{})
		for _, x := range viper.GetStringSlice("channels") {
			newchans[x] = struct{}{}
		}

		oldchans := make(map[string]struct{})
		for _, x := range channels {
			oldchans[x] = struct{}{}
		}

		// channels in newchans, but not oldchans are joined
		for n, _ := range newchans {
			if _, exists := oldchans[n]; !exists {
				join = append(join, n)
			}
		}

		for n, _ := range oldchans {
			if _, exists := newchans[n]; !exists {
				part = append(join, n)
			}
		}

		log.Printf("Leaving %v channels", part)
		for _, c := range part {
			bot.SendRawf("PART %s", c)
		}

		log.Printf("Joining following channels : %v", join)
		for _, c := range join {
			bot.Join(c)
		}

		channels = viper.GetStringSlice("channels")
		log.Print("Channel's config changed")
	}

	if nickname != viper.GetString("nickname") {
		nickname = viper.GetString("nickname")
		bot.Nick(nickname)
		log.Printf("Updating Nick to %s", nickname)
	}

	if probability != viper.GetInt("probability") {
		oldprob := probability
		probability = viper.GetInt("probability")

		log.Printf("Updating probability of kicking from %d to %d",
			oldprob, probability)
	}

	return nil
}
