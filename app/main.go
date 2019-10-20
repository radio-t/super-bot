package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"
	"github.com/sromku/go-gitter"

	"git.umputun.com/radio-t/gitter-rt-bot/app/bot"
	"git.umputun.com/radio-t/gitter-rt-bot/app/events"
	"git.umputun.com/radio-t/gitter-rt-bot/app/reporter"
)

var opts struct {
	GitterToken  string           `short:"t" long:"token" env:"GITTER_TOKEN" description:"gitter token" required:"true"`
	RoomID       string           `short:"r" long:"room" env:"GITTER_ROOM" description:"gitter room" default:"57141ba3187bb6f0eadfea6b"`
	RtjcPort     int              `short:"p" long:"port" env:"RTJC_PORT" default:"18001" description:"rtjc port room"`
	LogsPath     string           `short:"l" long:"logs" env:"GITTER_LOGS" default:"logs" description:"path to logs"`
	SuperUsers   events.SuperUser `long:"super" description:"super-users"`
	MashapeToken string           `long:"mashape" env:"MASHAPE_TOKEN" description:"mashape token"`
	SysData      string           `long:"sys-data" env:"SYS_DATA" default:"data" description:"location of sys data"`
	ExportNum    int              `long:"export-num" description:"show number for export"`
	ExportPath   string           `long:"export-path" default:"logs" description:"path to export directory"`
	ExportDay    int              `long:"export-day" description:"day in yyyymmdd"`
	TemplateFile string           `long:"export-template" default:"logs.html" description:"path to template file"`
	ExternalAPI  string           `long:"external-api" default:"https://bot.radio-t.com" description:"external api"`
	Dbg          bool             `long:"dbg" description:"debug mode"`
}

var revision = "local"

func main() {
	fmt.Printf("Radio-T bot for Gitter, %s\n", revision)
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	setupLog(opts.Dbg)
	log.Printf("[INFO] super users: %v", opts.SuperUsers)

	if opts.ExportNum > 0 {
		export()
		return
	}
	rand.Seed(int64(time.Now().Nanosecond()))
	api := gitter.New(opts.GitterToken)

	go events.Rtjc{Port: opts.RtjcPort, Gitter: api, RoomID: opts.RoomID}.Listen()

	multiBot := bot.MultiBot{
		bot.NewSys(opts.SysData),
		bot.NewVotes(opts.SuperUsers),
		bot.NewNews("https://news.radio-t.com/api"),
		bot.NewAnecdote(),
		bot.NewStackOverflow(),
		bot.NewDuck(opts.MashapeToken),
	}

	term := events.Terminator{
		BanDuration:   time.Minute * 10,
		BanPenalty:    2,
		AllowedPeriod: time.Minute * 2,
		Exclude:       opts.SuperUsers,
	}

	autoban := events.AutoBan{
		RoomID:      opts.RoomID,
		GitterToken: opts.GitterToken,
		MaxMsgSize:  1000,
		MsgsPerSec:  5,
		DupsPerSec:  3,
	}

	eListener := events.Listener{
		Terminator: term,
		AutoBan:    autoban,
		Reporter:   reporter.NewLogger(opts.LogsPath),
		API:        api,
		RoomID:     opts.RoomID,
		Bots:       multiBot,
	}

	eListener.Do()
}

func export() {
	log.Printf("[INFO] export mode, destination=%s, template=%s", opts.ExportPath, opts.TemplateFile)
	params := reporter.ExporterParams{
		InputRoot:    opts.LogsPath,
		OutputRoot:   opts.ExportPath,
		TemplateFile: opts.TemplateFile,
		SuperUsers:   opts.SuperUsers,
	}
	reporter.NewExporter(params).Export(opts.ExportNum, opts.ExportDay)
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
