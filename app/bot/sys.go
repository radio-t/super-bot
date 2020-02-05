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
	basic        map[string]string
	basicDesc    map[string]string
	com          []string // track order while ranging basicDesc
	dataLocation string
}

// NewSys makes new sys bot and load data to []say and basic map
func NewSys(dataLocation string) *Sys {
	log.Printf("[INFO] created sys bot, data location=%s", dataLocation)
	res := Sys{dataLocation: dataLocation}
	res.loadBasicData()
	res.loadSayData()
	return &res
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

	if val, found := p.basic[strings.ToLower(msg.Text)]; found {
		return Response{Text: val, Send: true}
	}

	return Response{}
}

func (p *Sys) loadBasicData() {
	bdata, err := readLines(p.dataLocation + "/basic.data")
	if err != nil {
		log.Fatalf("[FATAL] can't load basic.data, %v", err)
	}

	basic := make(map[string]string)
	basicDesc := make(map[string]string)
	com := make([]string, 0)
	for _, line := range bdata {
		elems := strings.Split(line, "|")
		// add '/say;say' to basicDesc
		if len(elems) == 2 {
			for _, key := range strings.Split(elems[0], ";") {
				basicDesc[key] = elems[1]
				com = append(com, key)
			}
			continue
		}
		if len(elems) != 3 {
			log.Printf("[DEBUG] bad format %s, ignored", line)
			continue
		}
		for _, key := range strings.Split(elems[0], ";") {
			basic[key] = elems[2]
			basicDesc[key] = elems[1]
			com = append(com, key)
		}
	}
	p.basic = basic
	p.basicDesc = basicDesc
	p.com = com
	log.Printf("[DEBUG] loaded basic set of responses, %v", basic)
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
	return p.com
}

func (p Sys) Help() (line string) {
	var s []string = p.ReactOn()
	// walk through all keys in p.com, group by response and add responce to each group
	for i := 0; i < len(s); i++ {
		var desc = p.basicDesc[s[i]]
		var a []string
		for j := i; j < len(s); j++ {
			if p.basicDesc[s[j]] != desc {
				break
			}
			a = append(a, s[j])
			i = j
		}
		line = line + "\n" + strings.Join(a, ", ") + " _- " + desc + "_"
	}
	return line
}
