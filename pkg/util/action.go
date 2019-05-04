package util

import (
	"fmt"
)

func Action(actionMessage string, f func() error) error {
	fmt.Println(actionMessage)
	if err := f(); err != nil {
		return err
	} else {
		fmt.Println("\n  OK")
		fmt.Println()
	}
	return nil
}
