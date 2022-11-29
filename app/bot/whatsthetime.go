package bot

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type WhatsTheTime struct {
	hosts []host
}

type host struct {
	name string
	timezone string
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

func (p *WhatsTheTime) loadTimeData(dataLocation string) error {
	data, err := readLines(dataLocation + "/whatsthetime.data")
	if err != nil {
		return fmt.Errorf("can't load whatsthetime.data: %w", err)
	}

	for _, line := range data {
		elems := strings.Split(line, "|")
		if len(elems) != 2 {
			log.Printf("[DEBUG] bad format %s, ignored", line)
			continue
		}
		host := host{
			name: elems[0],
			timezone: elems[1],
		}
		p.hosts = append(p.hosts, host)
		log.Printf("[DEBUG] loaded basic response, %s, %s", host.name, host.timezone)
	}
	return nil
}

func (w *WhatsTheTime) OnMessage(msg Message) (response Response) {
	if !contains(w.ReactOn(), msg.Text) {
		return Response{}
	}

	responseString := ""
	for _, host := range w.hosts {
		location, err := time.LoadLocation(host.timezone)
		if err != nil {
			log.Printf("[ERROR] can't load location for %s: %s", host.timezone, err)
			continue
		}
		responseString += fmt.Sprintf("У %s сейчас %s\n", host.name, time.Now().In(location).Format("15:04"))
	}
	return Response{Text: responseString, Send: true}
}

func (p *WhatsTheTime) ReactOn() []string {
	return []string{"время!", "time!", "который час?"}
}

func (p *WhatsTheTime) Help() (line string) {
	return genHelpMsg(p.ReactOn(), "подcкажет время у ведущих")
}