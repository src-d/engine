package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/src-d/sourced-ce/cmd/sourced/compose"
	"github.com/src-d/sourced-ce/cmd/sourced/compose/workdir"
	"github.com/src-d/sourced-ce/cmd/sourced/dir"
)

// The service name used in docker-compose.yml for the srcd/sourced-ui image
const containerName = "sourced-ui"

type webCmd struct {
	Command `name:"web" short-description:"Open the web interface in your browser." long-description:"Open the web interface in your browser, by default at: http://127.0.0.1:8088 user:admin pass:admin"`
}

func (c *webCmd) Execute(args []string) error {
	return OpenUI(2 * time.Second)
}

func init() {
	rootCmd.AddCommand(&webCmd{})
}

func openUI(address string) error {
	// docker-compose returns 0.0.0.0 which is correct for the bind address
	// but incorrect as connect address
	url := fmt.Sprintf("http://%s", strings.Replace(address, "0.0.0.0", "127.0.0.1", 1))

	for {
		client := http.Client{Timeout: time.Second}
		if _, err := client.Get(url); err == nil {
			break
		}

		time.Sleep(1 * time.Second)
	}

	if err := browser.OpenURL(url); err != nil {
		return errors.Wrap(err, "could not open the browser")
	}

	return nil
}

func checkFailFast(stdout *bytes.Buffer) (bool, error) {
	err := compose.RunWithIO(context.Background(),
		os.Stdin, stdout, nil, "port", containerName, "8088")
	if workdir.ErrMalformed.Is(err) || dir.ErrNotExist.Is(err) || dir.ErrNotValid.Is(err) {
		return true, err
	}

	if err != nil {
		return false, err
	}

	return false, nil
}

func waitForContainer(stdout *bytes.Buffer) {
	for {
		if err := compose.RunWithIO(context.Background(),
			os.Stdin, stdout, nil, "port", containerName, "8088"); err == nil {
			break
		}

		time.Sleep(1 * time.Second)
	}
}

// OpenUI opens the browser with the UI.
func OpenUI(timeout time.Duration) error {
	var stdout bytes.Buffer
	failFast, err := checkFailFast(&stdout)
	if failFast {
		return err
	}

	ch := make(chan error)
	containerReady := err == nil

	go func() {
		if !containerReady {
			waitForContainer(&stdout)
		}

		address := strings.TrimSpace(stdout.String())
		if address == "" {
			ch <- fmt.Errorf("could not find the public port of %s", containerName)
			return
		}

		ch <- openUI(address)
	}()

	fmt.Println(`
Once source{d} is fully initialized, the UI will be available, by default at:
  http://127.0.0.1:8088
  user:admin
  pass:admin
	`)

	if timeout > 5*time.Second {
		stopSpinner := startSpinner("Initializing source{d}...")
		defer stopSpinner()
	}

	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("error opening the UI, the container is not running after %v", timeout)
	}
}

type spinner struct {
	msg      string
	charset  []int
	interval time.Duration

	stop chan bool
}

func startSpinner(msg string) func() {
	charset := []int{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	if runtime.GOOS == "windows" {
		charset = []int{'|', '/', '-', '\\'}
	}

	s := &spinner{
		msg:      msg,
		charset:  charset,
		interval: 200 * time.Millisecond,
		stop:     make(chan bool),
	}
	s.Start()

	return s.Stop
}

func (s *spinner) Start() {
	go s.printLoop()
}

func (s *spinner) Stop() {
	s.stop <- true
}

func (s *spinner) printLoop() {
	i := 0
	for {
		select {
		case <-s.stop:
			fmt.Println(s.msg)
			return
		default:
			char := string(s.charset[i%len(s.charset)])
			if runtime.GOOS == "windows" {
				fmt.Printf("\r%s %s", s.msg, char)
			} else {
				fmt.Printf("%s %s\n\033[A", s.msg, char)
			}

			time.Sleep(s.interval)
		}

		i++
		if len(s.charset) == i {
			i = 0
		}
	}
}
