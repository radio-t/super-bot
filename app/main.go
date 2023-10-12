package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/go-pkgz/requester"
	"github.com/go-pkgz/requester/middleware"
	"github.com/go-pkgz/requester/middleware/logger"
	"github.com/go-pkgz/syncs"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jessevdk/go-flags"
	"golang.org/x/time/rate"

	"github.com/radio-t/super-bot/app/bot"
	"github.com/radio-t/super-bot/app/bot/openai"
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

	SpamFilter struct {
		Enabled   bool          `long:"enabled" env:"ENABLED" description:"enable spam filter"`
		API       string        `long:"api" env:"CAS_API" default:"https://api.cas.chat" description:"CAS API"`
		TimeOut   time.Duration `long:"timeout" env:"TIMEOUT" default:"5s" description:"CAS timeout"`
		Samples   string        `long:"samples" env:"SAMPLES" default:"" description:"path to spam samples"`
		Threshold float64       `long:"threshold" env:"THRESHOLD" default:"0.5" description:"spam threshold"`
		Dry       bool          `long:"dry" env:"DRY" description:"dry mode, no bans"`
	} `group:"spam-filter" namespace:"spam-filter" env-namespace:"SPAM_FILTER"`

	OpenAI struct {
		AuthToken         string `long:"token" env:"AUTH_TOKEN" description:"OpenAI auth token"`
		MaxTokensResponse int    `long:"max-tokens" env:"MAX_TOKENS" default:"1000" description:"OpenAI max_tokens in response"`
		MaxTokensRequest  int    `long:"max-tokens-request" env:"MAX_TOKENS_REQUEST" default:"3000" description:"OpenAI max tokens in request"`
		MaxSymbolsRequest int    `long:"max-symbols-request" env:"MAX_SYMBOLS_REQUEST" default:"12000" description:"OpenAI max symbols in request for fallback logic"`
		Prompt            string `long:"prompt" env:"PROMPT" default:"" description:"OpenAI prompt"`

		EnableAutoResponse      bool `long:"auto-response" env:"AUTO_RESPONSE" description:"enable auto response from OpenAI"`
		HistorySize             int  `long:"history-size" env:"HISTORY_SIZE" default:"5" description:"OpenAI history size for context answers"`
		HistoryReplyProbability int  `long:"history-reply-probability" env:"HISTORY_REPLY_PROBABILITY" default:"10" description:"percentage of the probability to reply with history (0%-100%)"`

		Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"120s" description:"OpenAI timeout in seconds"`
	} `group:"openai" namespace:"openai" env-namespace:"OPENAI"`

	RemarkAPI            string `long:"remark-api" env:"REMARK_API" default:"https://remark42.radio-t.com/api/v1/find" description:"Remark API"`
	UreadabilityAPI      string `long:"ur-api" env:"UREADABILITY_API" default:"https://ureadability.radio-t.com/api/content/v1/parser" description:"uReadability API"`
	UreadabilityToken    string `long:"ur-token" env:"UREADABILITY_TOKEN" default:"undefined" description:"uReadability token"`
	SummarizerThreadsNum int    `long:"summarizer-threads" env:"SUMMARIZER_THREADS" default:"5" description:"Number of threads in summarizer"`

	RtjcParams struct {
		SwgSize   int   `long:"swg-size" env:"SWG_SIZE" default:"10" description:"Rtjc sized waiting group size"`
		RateSec   int64 `long:"rate-sec" env:"RATE_SEC" default:"8" description:"Rtjc submit rate limit seconds between submits"`
		RateBurst int   `long:"rate-burst" env:"RATE_BURST" default:"5" description:"Rtjc submit rate limit burst"`
	} `group:"rtjc" namespace:"rtjc" env-namespace:"RTJC"`

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
	log.Printf("[DEBUG] opts: %+v", opts)
	if opts.ExportNum > 0 {
		export()
		return
	}

	tbAPI, err := tbapi.NewBotAPI(opts.Telegram.Token)
	if err != nil {
		log.Fatalf("[ERROR] can't make telegram bot, %v", err)
	}
	tbAPI.Debug = opts.Dbg

	httpClient := &http.Client{Timeout: 5 * time.Second}
	// 5 seconds is not enough for OpenAI requests
	httpClientOpenAI := makeOpenAIHttpClient()
	openAIBot := openai.NewOpenAI(openai.Params{
		AuthToken:               opts.OpenAI.AuthToken,
		MaxTokensResponse:       opts.OpenAI.MaxTokensResponse,
		MaxTokensRequest:        opts.OpenAI.MaxTokensRequest,
		MaxSymbolsRequest:       opts.OpenAI.MaxSymbolsRequest,
		Prompt:                  opts.OpenAI.Prompt,
		HistorySize:             opts.OpenAI.HistorySize,
		HistoryReplyProbability: opts.OpenAI.HistoryReplyProbability,
		EnableAutoResponse:      opts.OpenAI.EnableAutoResponse,
	}, httpClientOpenAI, opts.SuperUsers)

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
		bot.NewPrepPost(httpClient, "https://radio-t.com/site-api", 5*time.Minute),
		bot.NewWTF(time.Hour*24, 7*time.Hour*24, opts.SuperUsers),
		bot.NewBanhammer(tbAPI, opts.SuperUsers, 5000),
		bot.NewWhen(),
		openAIBot,
	}

	if opts.SpamFilter.Enabled {
		log.Printf("[INFO] spam filter enabled, api=%s, timeout=%s, dry=%v",
			opts.SpamFilter.API, opts.SpamFilter.TimeOut, opts.SpamFilter.Dry)
		httpCasClient := &http.Client{Timeout: opts.SpamFilter.TimeOut}
		multiBot = append(multiBot, bot.NewSpamCasFilter(opts.SpamFilter.API, httpCasClient, opts.SuperUsers, opts.SpamFilter.Dry))
		if opts.SpamFilter.Samples != "" {
			spamFh, err := os.Open(opts.SpamFilter.Samples)
			if err != nil {
				log.Fatalf("[ERROR] failed to open spam samples file %s, %v", opts.SpamFilter.Samples, err)
			}
			spamContent, ere := io.ReadAll(spamFh)
			if ere != nil {
				log.Fatalf("[ERROR] failed to read spam samples file %s, %v", opts.SpamFilter.Samples, err)
			}
			spamReaderLocal := bytes.NewReader(spamContent)
			// spamReaderAI := bytes.NewReader(spamContent)
			multiBot = append(multiBot,
				bot.NewSpamLocalFilter(spamReaderLocal, opts.SpamFilter.Threshold, opts.SuperUsers, opts.SpamFilter.Dry),
				// bot.NewSpamOpenAIFilter(spamReaderAI, openAIBot, opts.OpenAI.MaxSymbolsRequest, opts.SuperUsers, opts.SpamFilter.Dry),
			)
		}
	} else {
		log.Print("[INFO] spam filter disabled")
	}

	if sb, err := bot.NewSys(opts.SysData); err == nil {
		multiBot = append(multiBot, sb)
	} else {
		log.Printf("[ERROR] failed to load sysbot, %v", err)
	}

	if wttb, err := bot.NewWhatsTheTime(opts.SysData); err == nil {
		multiBot = append(multiBot, wttb)
	} else {
		log.Printf("[ERROR] failed to load whats the time bot, %v", err)
	}

	allActivityTerm := events.Terminator{
		BanDuration:   time.Minute * 5,
		BanPenalty:    10,
		AllowedPeriod: time.Second * 60,
		Exclude:       opts.SuperUsers,
	}

	botsActivityTerm := events.Terminator{
		BanDuration:   time.Minute * 15,
		BanPenalty:    3,
		AllowedPeriod: time.Minute * 5,
		Exclude:       opts.SuperUsers,
	}

	botsAllUsersActivityTerm := events.Terminator{
		BanDuration:   time.Minute * 5,
		BanPenalty:    5,
		AllowedPeriod: time.Minute * 5,
		Exclude:       opts.SuperUsers,
	}

	tgListener := events.TelegramListener{
		TbAPI:                  tbAPI,
		AllActivityTerm:        allActivityTerm,
		BotsActivityTerm:       botsActivityTerm,
		OverallBotActivityTerm: botsAllUsersActivityTerm,
		MsgLogger:              reporter.NewLogger(opts.LogsPath),
		Bots:                   multiBot,
		Group:                  opts.Telegram.Group,
		Debug:                  opts.Dbg,
		IdleDuration:           opts.IdleDuration,
		SuperUsers:             opts.SuperUsers,
	}

	remarkClient := openai.RemarkClient{
		Client: httpClient,
		API:    opts.RemarkAPI,
	}

	uKeeperClient := openai.UKeeperClient{
		Client: httpClient,
		API:    opts.UreadabilityAPI,
		Token:  opts.UreadabilityToken,
	}

	summarizer := openai.NewSummarizer(
		openAIBot,
		remarkClient,
		uKeeperClient,
		opts.SummarizerThreadsNum,
		opts.Dbg,
	)

	rtjc := events.Rtjc{
		Port:            opts.RtjcPort,
		Submitter:       &tgListener,
		Summarizer:      summarizer,
		Swg:             syncs.NewSizedGroup(opts.RtjcParams.SwgSize),
		SubmitRateBurst: opts.RtjcParams.RateBurst,
		SubmitRateLimit: rate.Limit(1 / float64(opts.RtjcParams.RateSec)),
	}
	go rtjc.Listen(ctx)

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

// makeOpenAIHttpClient creates http client with retry middleware
func makeOpenAIHttpClient() *http.Client {
	rpt := repeater.NewDefault(10, time.Second*5)
	lg := logger.New(lgr.Std)
	return requester.New(http.Client{Timeout: opts.OpenAI.Timeout}).With(middleware.Repeater(rpt), lg.Middleware).Client()
}

func setupLog(dbg bool) {
	logOpts := []lgr.Option{lgr.Msec, lgr.LevelBraces, lgr.StackTraceOnError}
	if dbg {
		logOpts = []lgr.Option{lgr.Debug, lgr.CallerFile, lgr.CallerFunc, lgr.Msec, lgr.LevelBraces, lgr.StackTraceOnError}
	}
	lgr.SetupStdLogger(logOpts...)
}
