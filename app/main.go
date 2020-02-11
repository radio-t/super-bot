package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/go-pkgz/lgr"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/umputun/go-flags"

	"github.com/radio-t/super-bot/app/bot"
	"github.com/radio-t/super-bot/app/events"
	"github.com/radio-t/super-bot/app/reporter"
	"github.com/radio-t/super-bot/app/storage"
)

var opts struct {
	Telegram struct {
		Token   string        `long:"token" env:"TOKEN" description:"telegram bot token" default:"test"`
		Group   string        `long:"group" env:"GROUP" description:"group name/id" default:"test"`
		Timeout time.Duration `long:"timeout" env:"TIMEOUT" description:"http client timeout for getting files from Telegram" default:"30s"`
	} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`

	RtjcPort             int              `short:"p" long:"port" env:"RTJC_PORT" default:"18001" description:"rtjc port room"`
	LogsPath             string           `short:"l" long:"logs" env:"TELEGRAM_LOGS" default:"logs" description:"path to logs"`
	SuperUsers           events.SuperUser `long:"super" description:"super-users"`
	MashapeToken         string           `long:"mashape" env:"MASHAPE_TOKEN" description:"mashape token"`
	SysData              string           `long:"sys-data" env:"SYS_DATA" default:"data" description:"location of sys data"`
	NewsArticles         int              `long:"max-articles" env:"MAX_ARTICLES" default:"5" description:"max number of news articles"`
	IdleDuration         time.Duration    `long:"idle" env:"IDLE" default:"30s" description:"idle duration"`
	ExportNum            int              `long:"export-num" description:"show number for export"`
	ExportPath           string           `long:"export-path" default:"logs" description:"path to export directory"`
	ExportDay            int              `long:"export-day" description:"day in yyyymmdd"`
	TemplateFile         string           `long:"export-template" default:"logs.html" description:"path to template file"`
	ExportBroadcastUsers events.SuperUser `long:"broadcast" description:"broadcast-users"`

	Dbg bool `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "local"

func main() {
	ctx := context.TODO()

	fmt.Printf("radio-t bot, %s\n", revision)
	if _, err := flags.Parse(&opts); err != nil {
		log.Printf("[ERROR] failed to parse flags: %v", err)
		os.Exit(1)
	}

	setupLog(opts.Dbg)
	log.Printf("[INFO] super users: %v", opts.SuperUsers)

	if opts.ExportNum > 0 {
		export()
		return
	}
	rand.Seed(int64(time.Now().Nanosecond()))

	httpClient := &http.Client{Timeout: 5 * time.Second}
	multiBot := bot.MultiBot{
		bot.NewBroadcastStatus(
			ctx,
			bot.BroadcastParams{
				URL:          "https://stream.radio-t.com",
				PingInterval: 10 * time.Second,
				DelayToOff:   time.Minute,
				Client:       http.Client{Timeout: 5 * time.Second}}),
		bot.NewNews(httpClient, "https://news.radio-t.com/api", opts.NewsArticles),
		bot.NewAnecdote(httpClient),
		bot.NewStackOverflow(),
		bot.NewDuck(opts.MashapeToken, httpClient),
		bot.NewPodcasts(httpClient, "https://radio-t.com/site-api", 5),
	}

	if sb, err := bot.NewSys(opts.SysData); err == nil {
		multiBot = append(multiBot, sb)
	} else {
		log.Printf("[ERROR] failed to load sysbot, %v", err)
	}

	term := events.Terminator{
		BanDuration:   time.Minute * 10,
		BanPenalty:    3,
		AllowedPeriod: time.Second * 5,
		Exclude:       opts.SuperUsers,
	}

	tbApi, err := tbapi.NewBotAPI(opts.Telegram.Token)
	if err != nil {
		log.Fatalf("[ERROR] can't make telegram bot, %v", err)
	}
	tbApi.Debug = opts.Dbg

	tgListener := events.TelegramListener{
		TbAPI:           tbApi,
		AllActivityTerm: term,
		MsgLogger:       reporter.NewLogger(opts.LogsPath),
		Bots:            multiBot,
		Group:           opts.Telegram.Group,
		Debug:           opts.Dbg,
		IdleDuration:    opts.IdleDuration,
	}

	go events.Rtjc{Port: opts.RtjcPort, Submitter: &tgListener}.Listen(ctx)
	if err := tgListener.Do(ctx); err != nil {
		log.Fatalf("[ERROR] telegram listener failed, %v", err)
	}
}

func export() {
	log.Printf("[INFO] export mode, destination=%s, template=%s", opts.ExportPath, opts.TemplateFile)
	botAPI, err := tbapi.NewBotAPI(opts.Telegram.Token)
	if err != nil {
		log.Fatalf("[ERROR] telegram bot creation failed: %v", err)
	}
	botUser, err := botAPI.GetMe()
	if err != nil {
		log.Fatalf("[ERROR] failed to get bot username: %v", err)
	}

	fileRecipient := reporter.NewTelegramFileRecipient(botAPI, opts.Telegram.Timeout)

	exportNum := strconv.Itoa(opts.ExportNum)
	s, err := storage.NewLocal(
		opts.ExportPath+"/"+exportNum,
		exportNum,
	)
	if err != nil {
		log.Fatalf("[ERROR] storage creation failed: %v", err)
	}

	params := reporter.ExporterParams{
		InputRoot:    opts.LogsPath,
		OutputRoot:   opts.ExportPath,
		TemplateFile: opts.TemplateFile,
		BotUsername:  botUser.UserName,
		SuperUsers:   opts.SuperUsers,
		BroadcastUsers: events.SuperUser(
			append(
				[]string{botUser.UserName},
				opts.ExportBroadcastUsers...,
			),
		),
	}
	err = reporter.NewExporter(fileRecipient, s, params).Export(opts.ExportNum, opts.ExportDay)
	if err != nil {
		log.Fatalf("[ERROR] export failed: %v", err)
	}
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
