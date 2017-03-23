package speaker

import (
	"log"
	"os/exec"
)

func Say(what string) {
	cmd := exec.Command("say", what)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Wait()
}
