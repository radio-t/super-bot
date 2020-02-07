package bot

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenHelpMsg(t *testing.T) {
	require.Equal(t, "cmd `- description`\n", genHelpMsg([]string{"cmd"}, "description"))
}

func TestMultiBotHelp(t *testing.T) {
	b1 := &MockInterface{}
	b1.On("Help").Return("b1 help")
	b2 := &MockInterface{}
	b2.On("Help").Return("b2 help")

	require.Equal(t, "b1 helpb2 help", MultiBot{b1, b2}.Help())
}

func TestMultiBotReactsOnHelp(t *testing.T) {
	b := &MockInterface{}
	b.On("ReactOn").Return([]string{"help"})
	b.On("Help").Return("help")

	mb := MultiBot{b}
	resp := mb.OnMessage(Message{Text: "help"})

	require.True(t, resp.Send)
	require.Equal(t, "help", resp.Text)
}

func TestMultiBotCombinesAllBotResponses(t *testing.T) {
	msg := Message{Text: "cmd"}

	b1 := &MockInterface{}
	b1.On("ReactOn").Return([]string{"cmd"})
	b1.On("OnMessage", msg).Return(Response{
		Text: "b1 resp",
		Send: true,
	})
	b2 := &MockInterface{}
	b2.On("ReactOn").Return([]string{"cmd"})
	b2.On("OnMessage", msg).Return(Response{
		Text: "b2 resp",
		Send: true,
	})

	mb := MultiBot{b1, b2}
	resp := mb.OnMessage(msg)

	require.True(t, resp.Send)
	parts := strings.Split(resp.Text, "\n")
	require.Len(t, parts, 2)
	require.Contains(t, parts, "b1 resp")
	require.Contains(t, parts, "b2 resp")
}
