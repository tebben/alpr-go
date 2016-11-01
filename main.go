package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tebben/alpr-go/configuration"
	"github.com/tebben/alpr-go/models"
	"github.com/tebben/alpr-go/mqtt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
	"os"
	"os/signal"
	"syscall"
)

var (
	cmd 		*exec.Cmd
	config          configuration.Config
	mqttClient      models.MQTTClient
	newPlates       = make(map[string]models.Result)
	publishedPlates = make(map[string]models.Result)
	mutex           = &sync.Mutex{}
)

func main() {
	log.Println("Starting lpr-to-gost")

	cfgFlag := flag.String("config", "config.yaml", "path of the config file")
	flag.Parse()

	cfg := *cfgFlag

	var err error
	config, err = configuration.GetConfig(cfg)
	if err != nil {
		log.Fatal("config read error: ", err)
		return
	}

	startMqtt()
	startAlpr()
	setupCleanup()
}

func setupCleanup(){
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(1)
	}()
}

func cleanup() {
	if err := cmd.Process.Kill(); err != nil {
		log.Fatal("failed to kill openalpr: ", err)
	}
}

// startMqtt starts a new mqtt client -> connect
func startMqtt() {
	mqttClient = mqtt.CreateMQTTClient(config.MQTT)
	mqttClient.Start()
}

// updatePlates runs al functions to keep track of new and published plates
func updatePlates(results []models.Result) {
	mutex.Lock()
	updatePublishedPlates(results)
	cleanPublishedPlates()
	updateNewPlates(results)
	checkPublishNewPlates()
	mutex.Unlock()
}

// updatePublishedPlates looks if the new results contain a plate that is already published
// if found the LastSeen property will be updated
func updatePublishedPlates(results []models.Result) {
	for _, r := range results {
		if plate, ok := publishedPlates[r.Plate]; ok {
			plate.LastSeen = time.Now().UnixNano()
		}
	}
}

// cleanPublishedPlates checks if a published plate should considered as lost based on
// the LastSeen property and Lost time defined in the config
func cleanPublishedPlates() {
	for k, r := range publishedPlates {
		if time.Now().UnixNano()-r.LastSeen >= (config.Alpr.Lost * 1000000) {
			delete(publishedPlates, k)
		}
	}
}

// updateNewPlates checks if a new plate is found, if so it will be added to the newPlates map
// if already in the list but the new result is of a higher confidence then the plate will be updated
func updateNewPlates(results []models.Result) {
	for _, r := range results {
		// plate is not in the published map, process it
		if _, ok := publishedPlates[r.Plate]; !ok {
			// plate is in new map, update it if higher confidence
			// ToDo: some sort of check if the map contains an almost identical plate, if
			// found and new is higher confidence remove old one and use new one
			if p, ok := newPlates[r.Plate]; ok {
				if r.Confidence > p.Confidence {
					p.Confidence = r.Confidence

				}
			} else if r.Confidence >= config.Alpr.Confidence && len(r.Plate) == 6 {
				r.FirstSeen = time.Now().UnixNano()
				r.LastSeen = time.Now().UnixNano()
				newPlates[r.Plate] = r
			}
		}
	}
}

// checkPublishNewPlates checks if plates in the newPlates map are passed their ScanTime, if so they
// will be published to GOST and removed from the map and added o the published map
func checkPublishNewPlates() {
	toPublish := make([]models.Result, 0)
	for _, p := range newPlates {
		if time.Now().UnixNano()-p.FirstSeen >= (config.Alpr.ScanTime * 1000000) {
			toPublish = append(toPublish, p)
			publishedPlates[p.Plate] = p
			delete(newPlates, p.Plate)
		}
	}

	if len(toPublish) > 0 {
		go publishToGost(toPublish)
	}
}

// publishToGost publishes the numberplate result to a given stream configured trough the config file
func publishToGost(results []models.Result) {
	for _, r := range results {
		mqttClient.Publish(fmt.Sprintf("GOST/Datastreams(%v)/Observations", config.MQTT.StreamID), fmt.Sprintf("{\"result\": { \"plate\": \"%s\", \"confidence\": %v } }", r.Plate, r.Confidence), 0)

		fmt.Println(fmt.Sprintf("Plate published to GOST: %s - %v", r.Plate, r.Confidence))
	}
}

// startAlpr starts the alpr software and checks for number plates
func startAlpr() {
	ch := make(chan string)
	go func() {
		err := RunCommandCh(ch, "\r\n", config.Alpr.Location, "-c", "eu", "-j", config.Alpr.Stream)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// happens between every alpr frame check, +/- 80-250 ms
	for v := range ch {
		go func(msg string) {
			var r models.Response
			err := json.Unmarshal([]byte(msg), &r)
			if err != nil {
				fmt.Println("Parse error: ", err)
			} else {
				updatePlates(r.Results)
			}
		}(fmt.Sprintf("%s", v))
	}
}

// RunCommandCh runs an arbitrary command and streams output to a channnel.
func RunCommandCh(stdoutCh chan<- string, cutset string, command string, flags ...string) error {
	cmd = exec.Command(command, flags...)
	output, err := cmd.StdoutPipe()

	if err != nil {
		return fmt.Errorf("RunCommand: cmd.StdoutPipe(): %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("RunCommand: cmd.Start(): %v", err)
	}

	go func() {
		defer close(stdoutCh)
		for {
			buf := make([]byte, 2048) //ToDo: buffer big enough? 1024 is not enough, results in multiple lines when 1 or 2 plates captured, else try to capture more and check if json found
			n, err := output.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Fatal(err)
				}
				if n == 0 {
					break
				}
			}
			text := strings.TrimSpace(string(buf[:n]))
			for {
				// Take the index of any of the given cutset
				n := strings.IndexAny(text, cutset)
				if n == -1 {
					// If not found, but still have data, send it
					if len(text) > 0 {
						stdoutCh <- text
					}
					break
				}
				// Send data up to the found cutset
				stdoutCh <- text[:n]
				// If cutset is last element, stop there.
				if n == len(text) {
					break
				}
				// Shift the text and start again.
				text = text[n+1:]
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("RunCommand: cmd.Wait(): %v", err)
	}

	return nil
}
