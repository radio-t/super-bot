package bot

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiBotReactsOnHelp(t *testing.T) {
	b := &MockInterface{}
	b.On("ReactOn").Return([]string{"cmd1", "cmd2"})

	mb := MultiBot{b}
	resp := mb.OnMessage(Message{Text: "help"})

	require.True(t, resp.Send)
	require.Equal(t, "_cmd1 cmd2_", resp.Text)
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
