package bot

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestMultiBotReactsOnHelp(t *testing.T) {
	b := &MockInterface{}
	b.On("ReactOn").Return([]string{"cmd1", "cmd2"})

	mb := MultiBot{b}
	resp, send := mb.OnMessage(Message{Text: "help"})

	require.True(t, send)
	require.Equal(t, "_cmd1 cmd2_", resp)
}

func TestMultiBotCombinesAllBotResponses(t *testing.T) {
	msg := Message{Text: "cmd"}

	b1 := &MockInterface{}
	b1.On("ReactOn").Return([]string{"cmd"})
	b1.On("OnMessage", msg).Return("b1 resp", true)
	b2 := &MockInterface{}
	b2.On("ReactOn").Return([]string{"cmd"})
	b2.On("OnMessage", msg).Return("b2 resp", true)

	mb := MultiBot{b1, b2}
	resp, send := mb.OnMessage(msg)

	require.True(t, send)
	parts := strings.Split(resp, "\n")
	require.Len(t, parts, 2)
	require.Contains(t, parts, "b1 resp")
	require.Contains(t, parts, "b2 resp")
}
