package log

import (
	"fmt"
	"os"
)

func Stderr(msg string) {
	fmt.Fprintln(os.Stderr, "gas: "+msg)
}
