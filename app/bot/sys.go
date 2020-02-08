package bot

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"

	log "github.com/go-pkgz/lgr"
)

// Sys implements basic bot function to responds on ping and others from basic.data file.
// also reacts and say! with keys/values from say.data file
type Sys struct {
	say          []string
	dataLocation string
	SysBots      []SysCommand
}
// SysCommand hold one type commands from basic.data
type SysCommand struct {
	commands    []string
	description string
	message     string
}

// NewSys makes new sys bot and load data to []say and basic map
func NewSys(dataLocation string) *Sys {
	log.Printf("[INFO] created sys bot, data location=%s", dataLocation)
	res := Sys{dataLocation: dataLocation}
	res.loadBasicData()
	res.loadSayData()
	return &res
}

// Help returns help message
func (p Sys) Help() (line string) {
	for _, com := range p.SysBots {
		line += genHelpMsg(com.commands, com.description)
	}
	return line
}

// OnMessage implements bot.Interface
func (p Sys) OnMessage(msg Message) (response Response) {
	if !contains(p.ReactOn(), msg.Text) {
		return Response{}
	}

	if msg.Text == "say!" || msg.Text == "/say" {
		if p.say != nil && len(p.say) > 0 {
			return Response{
				Text: fmt.Sprintf("_%s_", p.say[rand.Intn(len(p.say))]),
				Send: true,
			}
		}
		return Response{}
	}

	for _, bot := range p.SysBots {
		if found := contains(bot.commands, strings.ToLower(msg.Text)); found {
			return Response{Text: bot.message, Send: true}
		} 
	}

	return Response{}
}

func (p *Sys) loadBasicData() {
	bdata, err := readLines(p.dataLocation + "/basic.data")
	if err != nil {
		log.Fatalf("[FATAL] can't load basic.data, %v", err)
	}

	for _, line := range bdata {
		elems := strings.Split(line, "|")		
		if len(elems) != 3 {
			log.Printf("[DEBUG] bad format %s, ignored", line)
			continue
		}
		sysCommand := SysCommand{
			description: elems[1],
			message: elems[2],
			commands: strings.Split(elems[0], ";"),
		}
		p.SysBots = append(p.SysBots, sysCommand)
		log.Printf("[DEBUG] loaded basic response, %v, %s", sysCommand.commands, sysCommand.message)
	}
}

func (p *Sys) loadSayData() {
	say, err := readLines(p.dataLocation + "/say.data")
	if err != nil {
		log.Printf("[WARN] can't load say.data - %v", err)
		return
	}
	p.say = say
	log.Printf("[DEBUG] loaded say.data, %d records", len(say))
}

func readLines(path string) ([]string, error) {

	data, err := ioutil.ReadFile(path) // nolint
	if err != nil {
		log.Printf("[WARN] can't load data from %s,  %v", path, err)
		return nil, err
	}

	return strings.Split(string(data), "\n"), nil
}

// ReactOn keys
func (p Sys) ReactOn() []string {
	res := make([]string, 0)
	for _, bot := range p.SysBots {
		res = append(bot.commands, res...)
	}
	return res
}
