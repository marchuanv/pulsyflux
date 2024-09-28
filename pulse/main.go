package main

import (
	"fmt"
	"uuid"
)

func main() {
	fmt.Println("running")
	instance := uuid.New()
	str, error := instance.ToString()
	if error != nil {
		fmt.Println(error)
	} else {
		fmt.Println(str)
	}
}
