package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhatsTheTime_buildResponseText(t *testing.T) {
	t.Parallel()

	table := []struct {
		in  Host
		exp string
	}{
		{
			in:  Host{Name: `Ksenks`, Timezone: `America/Los_Angeles`},
			exp: "У Ksenks сейчас 12:20\n",
		},
		{
			in:  Host{Name: "Umputun", Timezone: "America/Chicago"},
			exp: "У Umputun сейчас 14:20\n",
		},
		{
			in:  Host{Name: "Bobuk", Timezone: "Europe/Kiev"},
			exp: "У Bobuk сейчас 23:20\n",
		},
		{
			in:  Host{Name: "Gray", Timezone: "Europe/Kiev"},
			exp: "У Gray сейчас 23:20\n",
		},
		{
			in:  Host{Name: "Alek.sys", Timezone: "Europe/London"},
			exp: "У Alek.sys сейчас 21:20\n",
		},
	}

	mockTime := time.Date(1970, 1, 1, 20, 20, 0, 0, time.UTC)
	for _, row := range table {
		t.Run("", func(t *testing.T) {
			res := buildResponseText(mockTime, []Host{row.in})
			assert.Equal(t, row.exp, res)
		})
	}
}

func TestWhatsTheTime_Help(t *testing.T) {
	b, err := NewWhatsTheTime("./../../data")
	require.NoError(t, err)
	require.Equal(t, "время!, time!, который час? _– подcкажет время у ведущих_\n", b.Help())
}
