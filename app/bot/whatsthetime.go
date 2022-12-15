package bot

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"
)

// WhatsTheTime answers which time is on hosts timezones
// uses whatsthetime.data file as configuration
type WhatsTheTime struct {
	hosts []Host
}

// Host is structure with name and timezone
type Host struct {
	Name string
	Timezone string
}

// NewWhatsTheTime makes new What's The Time bot and load data to []hosts
func NewWhatsTheTime(dataLocation string) (*WhatsTheTime, error) {
	log.Printf("[INFO] created WhatstTheTime bot, data location=%s", dataLocation)
	res := WhatsTheTime{}
	if err := res.loadTimeData(dataLocation); err != nil {
		return nil, err
	}
	return &res, nil
}

func (w *WhatsTheTime) loadTimeData(dataLocation string) error {
	data, err := readLines(filepath.Join(dataLocation,"whatsthetime.data"))
	if err != nil {
		return fmt.Errorf("can't load whatsthetime.data: %w", err)
	}

	for _, line := range data {
		elems := strings.Split(line, "|")
		if len(elems) != 2 {
			log.Printf("[DEBUG] bad format %s, ignored", line)
			continue
		}
		host := Host{
			Name: elems[0],
			Timezone: elems[1],
		}
		w.hosts = append(w.hosts, host)
		log.Printf("[DEBUG] loaded basic response, %s, %s", host.Name, host.Timezone)
	}
	return nil
}

// OnMessage returns one entry
func (w *WhatsTheTime) OnMessage(msg Message) (response Response) {
	if !contains(w.ReactOn(), msg.Text) {
		return Response{}
	}

	return Response{
		Text: buildResponseText(time.Now(), w.hosts), 
		Send: true,
	}
}

func buildResponseText(now time.Time, hosts []Host) string {
	responseString := ""
	for _, host := range hosts {
		location, err := time.LoadLocation(host.Timezone)
		if err != nil {
			log.Printf("[DEBUG] can't load location for %s: %s", host.Timezone, err)
			continue
		}
		responseString += fmt.Sprintf("У %s сейчас %s\n", host.Name, now.In(location).Format("15:04"))
	}
	return responseString
}

// ReactOn returns reaction keys
func (w *WhatsTheTime) ReactOn() []string {
	return []string{"время!", "time!", "который час?"}
}

// Help returns help message
func (w *WhatsTheTime) Help() (line string) {
	return genHelpMsg(w.ReactOn(), "подcкажет время у ведущих")
}