package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func init() {
}

// stringSlices args
type stringSlices []string

// Implement String for flags
func (f *stringSlices) String() string {
	return fmt.Sprintf("%v", *f)
}

// Implement Set for flags
func (f *stringSlices) Set(file string) error {
	*f = append(*f, file)
	return nil
}

// watchFile will look for ModTime/Size changes
func watchFile(filePath string, ch chan bool) {
	initialStat, err := os.Stat(filePath)
	if err != nil {
		log.Errorf("Cannot Stat on file %v: %v", filePath, err)
	}
	for {
		stat, err := os.Stat(filePath)
		if err != nil {
			log.Warnf("Cannot Stat on file %v: %v. Will retry in 10s.", filePath, err)
			time.Sleep(10 * time.Second)
			continue
		}

		if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
			log.Infof("-> File %v has changed", filePath)
			ch <- true
			initialStat, err = os.Stat(filePath)
			if err != nil {
				log.Errorf("Cannot Stat on file %v: %v", filePath, err)
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func watchFiles(files []string, ch chan bool) {
	toWatch := []string{}
	// If we have a folder as file we will take all files in it.
	for _, f := range files {
		if info, err := os.Stat(f); err == nil && info.IsDir() {
			fl, _ := ioutil.ReadDir(f)
			for _, i := range fl {
				toWatch = append(toWatch, f+"/"+i.Name())
			}
		} else if _, err := os.Stat(f); err == nil {
			toWatch = append(toWatch, f)
		} else {
			log.Warnf("File %v doesn't exist.. skipping...", f)
		}
	}

	if len(toWatch) == 0 {
		log.Fatal("No files to watch... aborting")
	}

	log.Infof("Watching changes on file: %v", strings.Join(toWatch, ", "))
	for _, file := range toWatch {
		go func(file string) { watchFile(file, ch) }(file)
	}
}

var myfiles stringSlices
var myargs stringSlices
var log = logrus.New()

func main() {

	log.Formatter = new(prefixed.TextFormatter)

	flag.Var(&myfiles, "f", "List of files to watch.")
	cmd := flag.String("c", "", "The command to run.")
	flag.Var(&myargs, "a", "The args to pass to the command.")

	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}
	var c *exec.Cmd

	// watchFiles watching for files

	// Create go routines to watch files, results will be sent trough a channel.
	ch := make(chan bool, 1)
	watchFiles(myfiles, ch)

	if *cmd != "" {
		// Fork the run
		go func() {
			for {
				log.Infof("-> Executing: %v with args %v", *cmd, myargs)
				c = exec.Command(*cmd, myargs...)
				var stderr bytes.Buffer

				c.Stderr = &stderr

				cmdReader, err := c.StdoutPipe()
				if err != nil {
					log.Fatal("Error creating StdoutPipe for Cmd: ", err)
				}

				scanner := bufio.NewScanner(cmdReader)
				go func() {
					for scanner.Scan() {
						log.WithField("prefix", "[Process]").Infof("%s", scanner.Text())
					}
				}()
				// Will block
				err = c.Run()
				if err != nil && err.Error() != "signal: killed" {
					log.Fatalf("Process exited with error: %v, %v", err, stderr.String())
				} else if err == nil {
					log.Fatal("Process exited with no error. Aborting...")
				}
				log.Infof("-> Command terminated with code %v", err)
			}
		}()

		// List on result chan in a runloop
		for {
			// Will block until a result is added.
			<-ch
			log.Info("-> Restarting Command.")
			c.Process.Kill()
		}
	}

}
