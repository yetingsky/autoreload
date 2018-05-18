package main

import (
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var file string
var args []string

func main() {
	if len(os.Args) < 2 {
		log.Fatal("at least two args are required")
	}

	file = os.Args[1]
	args = os.Args[2:]
	log.Printf("main file: %s", file)
	log.Printf("args: %s", args)

	cmd := run()
	go scanChanges(".", []string{}, func(path string) {
		kill(cmd)
		cmd = run()
	})

	sysStop := make(chan os.Signal, 1)
	signal.Notify(sysStop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	select {
	case <-sysStop:
		kill(cmd)
	}
}

func build() error {
	command := exec.Command("go", "build", "-o", "proc", file)

	output, err := command.CombinedOutput()
	if !command.ProcessState.Success() {
		err = errors.New(string(output))
	}

	return err
}

func run() *exec.Cmd {
	if err := build(); err != nil {
		log.Print(err)
	}

	cmd := exec.Command("./proc", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	time.Sleep(250 * time.Millisecond)
	return cmd
}

func kill(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		log.Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
	}
	return nil
}

var startTime = time.Now()

func scanChanges(watchPath string, excludeDirs []string, callback func(path string)) {
	for {
		filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
			if path == ".git" && info.IsDir() {
				return filepath.SkipDir
			}
			for _, x := range excludeDirs {
				if x == path {
					return filepath.SkipDir
				}
			}

			// ignore hidden files
			if filepath.Base(path)[0] == '.' {
				return nil
			}

			if filepath.Ext(path) == ".go" && info.ModTime().After(startTime) {
				callback(path)
				startTime = time.Now()
				return errors.New("done")
			}

			return nil
		})
		time.Sleep(500 * time.Millisecond)
	}
}
