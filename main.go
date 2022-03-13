package main

import (
	"fmt"

	"github.com/jaffee/commandeer/cobrafy"
)

func main() {
	err := cobrafy.Execute(NewSwitchHost())
	if err != nil {
		fmt.Println(err)
	}
}
