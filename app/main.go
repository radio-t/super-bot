package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/umputun/go-flags"

	"github.com/radio-t/gitter-rt-bot/app/bot"
	"github.com/radio-t/gitter-rt-bot/app/events"
	"github.com/radio-t/gitter-rt-bot/app/reporter"
)

var opts struct {
	Telegram struct {
		Token       string `long:"token" env:"TOKEN" description:"telegram bot token" required:"test"`
		Channel     string `long:"channel" env:"CHANNEL" description:"channel name/id" default:"test"`
		BotUserName string `long:"name" env:"NAME" description:"bot user name" default:"test"`
	} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`

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
	fmt.Printf("radio-t bot, %s\n", revision)
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

	multiBot := bot.MultiBot{
		bot.NewSys(opts.SysData),
		bot.NewVotes(opts.SuperUsers),
		bot.NewNews("https://news.radio-t.com/api"),
		bot.NewAnecdote(),
		bot.NewStackOverflow(),
		bot.NewDuck(opts.MashapeToken),
		bot.NewExcerpt("http://parser.ukeeper.com/api/content/v1/parser", "not-supported"),
	}

	term := events.Terminator{
		BanDuration:   time.Minute * 10,
		BanPenalty:    2,
		AllowedPeriod: time.Minute * 2,
		Exclude:       opts.SuperUsers,
	}

	channelID := opts.Telegram.Channel
	if _, err := strconv.ParseInt(channelID, 10, 64); err != nil {
		channelID = "@" + channelID // if channelID not a number enforce @ prefix
	}

	tgListener := events.TelegramListener{
		Terminator:  term,
		Reporter:    reporter.NewLogger(opts.LogsPath),
		Bots:        multiBot,
		ChannelID:   channelID,
		Token:       opts.Telegram.Token,
		BotUserName: opts.Telegram.BotUserName,
		Debug:       opts.Dbg,
	}

	ctx := context.TODO()
	go events.Rtjc{Port: opts.RtjcPort, Submitter: &tgListener}.Listen(ctx)
	if err := tgListener.Do(ctx); err != nil {
		log.Fatalf("[ERROR] telegram listener failed, %v", err)
	}
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
