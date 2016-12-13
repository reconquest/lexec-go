package lexec_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/reconquest/hierr-go"
	"github.com/reconquest/lexec-go"
)

func ExampleLoggedExec() {
	logger := log.New(os.Stdout, `LOG: `, 0)

	cmd := lexec.NewExec(
		lexec.Loggerf(logger.Printf),
		exec.Command(`wc`, `-l`),
	)

	cmd.SetStdin(bytes.NewBufferString("1\n2\n3\n"))

	err := cmd.Run()
	if err != nil {
		log.Fatalln(hierr.Errorf(
			err,
			`can't run example command`,
		))
	}

	stdout, err := ioutil.ReadAll(cmd.GetStdout())
	if err != nil {
		log.Fatalln(hierr.Errorf(
			err,
			`can't read command stdout`,
		))
	}

	fmt.Printf("OUT: %s\n", stdout)

	// Output:
	// LOG: {wc} <exec> ["wc" "-l"] start
	// LOG: {wc} <stdout> 3
	// LOG: {wc} <exec> ["wc" "-l"] exit code 0
	// OUT: 3
}
