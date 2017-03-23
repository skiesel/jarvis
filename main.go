package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/skiesel/jarvis/gortana"
	"github.com/skiesel/jarvis/speaker"
)

var (
	debugFile = flag.String("debug", "debug.log", "Debug log file to write log to.")
)

func main() {
	flag.Parse()

	debug, err := os.OpenFile(*debugFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	log.SetOutput(debug)

	defer debug.Close()

	inform := make(chan string, 100)

	gortana.Listen(inform)

	for {
		msg := <-inform
		fmt.Println(msg)
		gortana.Pause()
		if msg == "Bye!" {
			speaker.Say(msg)
		} else {
			speaker.Say("I think you said " + msg)
		}
		gortana.Resume()
	}
}
