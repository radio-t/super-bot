package main

import (
	"bytes"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/bot/openai"
	"github.com/radio-t/super-bot/app/bot/openai/jev"
)

func TestSetJev(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		enabled    bool
		timeout    time.Duration
		wantClient bool
		wantWarn   bool
		wantErr    bool
	}{
		{"enabled with key", "fake-jev-key", true, time.Second, true, false, false},
		{"disabled with key", "fake-jev-key", false, time.Second, true, false, false},
		{"enabled without key", "", true, time.Second, false, true, false},
		{"disabled without key", "", false, time.Second, false, false, false},
		{"zero timeout", "fake-jev-key", true, 0, false, false, true},
		{"negative timeout", "fake-jev-key", true, -time.Second, false, false, true},
		{"invalid timeout without key", "", false, 0, false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&output)
			t.Cleanup(func() { log.SetOutput(previous) })
			params := openai.Params{Model: "gpt-6-luna", EnableAutoResponse: tt.enabled,
				Jev: jev.New(jev.Params{Key: "old-fake-key", Model: "jev-old", Timeout: time.Second})}

			err := setJev(&params, jev.Params{Key: tt.key, Model: "jev-test", Timeout: tt.timeout}, tt.enabled)
			if tt.wantErr {
				require.ErrorContains(t, err, "timeout")
			} else {
				require.NoError(t, err)
			}
			if tt.wantClient {
				require.True(t, params.Jev != nil)
				assert.IsType(t, &jev.Client{}, params.Jev)
			} else {
				assert.True(t, params.Jev == nil)
			}
			if tt.wantWarn {
				assert.Contains(t, output.String(), "[WARN]")
				assert.Contains(t, output.String(), "JEV_KEY")
			} else {
				assert.Empty(t, output.String())
			}
			assert.NotContains(t, output.String(), "fake-jev-key")
			assert.NotContains(t, output.String(), "old-fake-key")
			assert.Equal(t, "gpt-6-luna", params.Model)
			assert.Equal(t, tt.enabled, params.EnableAutoResponse)
		})
	}
}

func TestRedactedOpts(t *testing.T) {
	previous := opts
	t.Cleanup(func() { opts = previous })
	opts.Telegram.Token = "fake-telegram-secret"
	opts.MashapeToken = "fake-mashape-secret"
	opts.OpenAI.AuthToken = "fake-openai-secret"
	opts.UreadabilityToken = "fake-ureadability-secret"
	opts.Jev.Key = "fake-jev-secret"
	opts.Jev.Model = "jev-public-model"
	opts.OpenAI.Model = "gpt-6-luna"
	want := opts

	dump := redactedOpts()
	for _, secret := range []string{opts.Telegram.Token, opts.MashapeToken, opts.OpenAI.AuthToken, opts.UreadabilityToken, opts.Jev.Key} {
		assert.NotContains(t, dump, secret)
	}
	assert.Contains(t, dump, "jev-public-model")
	assert.Contains(t, dump, "gpt-6-luna")
	assert.Equal(t, want, opts)
}

func TestJevFlags(t *testing.T) {
	for _, mode := range []string{"defaults", "environment", "flags", "removed probability"} {
		t.Run(mode, func(t *testing.T) {
			for _, key := range []string{"JEV_KEY", "JEV_MODEL", "JEV_TIMEOUT", "OPENAI_HISTORY_SIZE", "GO_FLAGS_COMPLETION"} {
				t.Setenv(key, "")
				require.NoError(t, os.Unsetenv(key))
			}
			for _, key := range []string{"TELEGRAM_TOKEN", "MASHAPE_TOKEN", "OPENAI_AUTH_TOKEN", "UREADABILITY_TOKEN"} {
				t.Setenv(key, "fake-test-secret")
			}
			args := []string{"--telegram.token=fake-telegram", "--telegram.group=test-group"}
			switch mode {
			case "environment":
				t.Setenv("JEV_KEY", "fake-jev-key")
				t.Setenv("JEV_MODEL", "jev-test")
				t.Setenv("JEV_TIMEOUT", "750ms")
			case "flags":
				t.Setenv("JEV_KEY", "overridden-fake-key")
				args = append(args, "--jev.key=fake-jev-key", "--jev.model=jev-test", "--jev.timeout=750ms")
			case "removed probability":
				args = append(args, "--openai.history-reply-probability=7")
			}
			parsed := opts
			_, err := flags.NewParser(&parsed, flags.None).ParseArgs(args)
			if mode == "removed probability" {
				require.ErrorContains(t, err, "unknown flag")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 10, parsed.OpenAI.HistorySize)
			if mode == "defaults" {
				assert.Empty(t, parsed.Jev.Key)
				assert.Equal(t, "jev-1.13.0", parsed.Jev.Model)
				assert.Equal(t, 2*time.Second, parsed.Jev.Timeout)
			} else {
				assert.Equal(t, "fake-jev-key", parsed.Jev.Key)
				assert.Equal(t, "jev-test", parsed.Jev.Model)
				assert.Equal(t, 750*time.Millisecond, parsed.Jev.Timeout)
			}
		})
	}
}
