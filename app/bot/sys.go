package bot

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

// Sys implements basic bot function to respond on ping and others from basic.data file.
// also, reacts on say! with keys/values from say.data file
type Sys struct {
	say          []string
	dataLocation string
	commands     []sysCommand
}

// sysCommand hold one type triggers from basic.data
type sysCommand struct {
	triggers    []string
	description string
	message     string
}

// NewSys makes new sys bot and load data to []say and basic map
func NewSys(dataLocation string) (*Sys, error) {
	log.Printf("[INFO] created sys bot, data location=%s", dataLocation)
	res := Sys{dataLocation: dataLocation}
	if err := res.loadBasicData(); err != nil {
		return nil, err
	}
	if err := res.loadSayData(); err != nil {
		return nil, err
	}
	rand.Seed(0)
	return &res, nil
}

// Help returns help message
func (p Sys) Help() (line string) {
	for _, c := range p.commands {
		line += genHelpMsg(c.triggers, c.description)
	}
	return line
}

// OnMessage implements bot.Interface
func (p Sys) OnMessage(msg Message) (response Response) {
	if !contains(p.ReactOn(), msg.Text) {
		return Response{}
	}

	if strings.EqualFold(msg.Text, "say!") {
		if p.say != nil && len(p.say) > 0 {
			return Response{
				Text: fmt.Sprintf("_%s_", EscapeMarkDownV1Text(p.say[rand.Intn(len(p.say))])), //nolint:gosec
				Send: true,
			}
		}
		return Response{}
	}

	for _, bot := range p.commands {
		if found := contains(bot.triggers, strings.ToLower(msg.Text)); found {
			return Response{Text: bot.message, Send: true}
		}
	}

	return Response{}
}

func (p *Sys) loadBasicData() error {
	bdata, err := readLines(filepath.Join(p.dataLocation, "basic.data"))
	if err != nil {
		return fmt.Errorf("can't load basic.data: %w", err)
	}

	for _, line := range bdata {
		elems := strings.Split(line, "|")
		if len(elems) != 3 {
			log.Printf("[DEBUG] bad format %s, ignored", line)
			continue
		}
		sysCommand := sysCommand{
			description: elems[1],
			message:     elems[2],
			triggers:    strings.Split(elems[0], ";"),
		}
		p.commands = append(p.commands, sysCommand)
		log.Printf("[DEBUG] loaded basic response, %v, %s", sysCommand.triggers, sysCommand.message)
	}
	return nil
}

func (p *Sys) loadSayData() error {
	say, err := readLines(filepath.Join(p.dataLocation, "say.data"))
	if err != nil {
		return err
	}
	p.say = say
	log.Printf("[DEBUG] loaded say.data, %d records", len(say))
	return nil
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %w", path, err)
	}
	defer f.Close() //nolint

	result := make([]string, 0)
	s := bufio.NewScanner(f)
	for s.Scan() {
		result = append(result, s.Text())
	}

	return result, nil
}

// ReactOn keys
func (p Sys) ReactOn() []string {
	res := make([]string, 0)
	for _, bot := range p.commands {
		res = append(bot.triggers, res...)
	}
	return res
}
