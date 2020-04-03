package bot

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
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
				Text: fmt.Sprintf("_%s_", p.say[rand.Intn(len(p.say))]),
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
	bdata, err := readLines(p.dataLocation + "/basic.data")
	if err != nil {
		return errors.Wrap(err, "can't load basic.data")
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
	say, err := readLines(p.dataLocation + "/say.data")
	if err != nil {
		return err
	}
	p.say = say
	log.Printf("[DEBUG] loaded say.data, %d records", len(say))
	return nil
}

func readLines(path string) ([]string, error) {

	data, err := ioutil.ReadFile(path) // nolint
	if err != nil {
		return nil, errors.Wrapf(err, "can't load sys data from %s", path)
	}

	return strings.Split(string(data), "\n"), nil
}

// ReactOn keys
func (p Sys) ReactOn() []string {
	res := make([]string, 0)
	for _, bot := range p.commands {
		res = append(bot.triggers, res...)
	}
	return res
}
