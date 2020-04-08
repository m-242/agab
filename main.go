package main

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/thoj/go-ircevent"
	"log"
	"math/rand"
)

var (
	channels    []string
	probability int
	server      string
	nickname    string
	reasons     []string
	tls         bool
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.SetDefault("reasons", [1]string{"You talk to much"})
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

	con := irc.IRC(nickname, "agab")
	con.UseTLS = viper.GetBool("tls")
	err = con.Connect(server)
	if err != nil {
		log.Panic("Failed connecting")
		return
	}
	con.AddCallback("001", func(e *irc.Event) {
		for _, channName := range channels {
			con.Join(channName)
		}
	})

	con.AddCallback("PRIVMSG", func(event *irc.Event) {
		nick := event.Nick
		chann := event.Arguments[0]

		if rand.Intn(probability) == 1 {
			req := fmt.Sprintf("KICK %s %s : %s",
				chann,
				nick,
				reasons[rand.Intn(len(reasons))])
			log.Printf("Kicked %s from %s", nick, chann)
			con.SendRaw(req)
		}
	})

	con.Loop()
}
