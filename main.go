package main

import (
  "flag"
  "fmt"

  "github.com/skiesel/jarvis/gortana"
  "github.com/skiesel/jarvis/speaker"
)

func main() {
  flag.Parse()

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
