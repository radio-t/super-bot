package bot

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVotes_StartVoting(t *testing.T) {
	su := &MockSuperUser{}
	su.On("IsSuper", mock.Anything).Return(true)

	b := NewVotes(su)
	resp := b.OnMessage(Message{Text: "++topic"})

	require.True(t, resp.Send)
	require.Equal(t, "голосование началось! (+1/-1) *topic*", resp.Text)
	require.True(t, b.started)
	require.Equal(t, "topic", b.topic)
}

func TestVotes_FinishVoteNoVotes(t *testing.T) {
	su := &MockSuperUser{}
	su.On("IsSuper", mock.Anything).Return(true)

	b := NewVotes(su)
	b.OnMessage(Message{Text: "++"})
	resp := b.OnMessage(Message{Text: "!!"})
	require.True(t, resp.Send)
	// strange behaviour
	require.Equal(t, "голосование завершено - __\n- *за: 100% (1)*\n- *против: 0% (0) *", resp.Text)
	require.False(t, b.started)
}

func TestVotes_Votes(t *testing.T) {
	su := &MockSuperUser{}
	su.On("IsSuper", mock.Anything).Return(true)

	b := NewVotes(su)
	b.OnMessage(Message{Text: "++"})
	b.OnMessage(Message{Text: "+1", From: User{Username: "u1"}})
	b.OnMessage(Message{Text: "+1", From: User{Username: "u2"}})
	b.OnMessage(Message{Text: "-1", From: User{Username: "u3"}})
	b.OnMessage(Message{Text: "-1", From: User{Username: "u4"}})
	resp := b.OnMessage(Message{Text: "!!"})
	require.True(t, resp.Send)
	require.Equal(t, "голосование завершено - __\n- *за: 50% (2)*\n- *против: 50% (2) *", resp.Text)
}

func TestVotes_FinishVote(t *testing.T) {
	su := &MockSuperUser{}
	su.On("IsSuper", mock.Anything).Return(true)

	b := NewVotes(su)
	b.OnMessage(Message{Text: "++"})

	b.votes["u1"] = true
	b.votes["u2"] = true
	b.votes["u3"] = false

	resp := b.OnMessage(Message{Text: "!!"})
	require.True(t, resp.Send)
	require.Equal(t, "голосование завершено - __\n- *за: 66% (2)*\n- *против: 33% (1) *", resp.Text)
	require.False(t, b.started)

	b.OnMessage(Message{Text: "+1", From: User{Username: "u1"}})
	b.OnMessage(Message{Text: "+1", From: User{Username: "u2"}})
	b.OnMessage(Message{Text: "-1", From: User{Username: "u2"}})
	b.OnMessage(Message{Text: "unexpected", From: User{Username: "u3"}})

	resp = b.OnMessage(Message{Text: "!!"})
	require.True(t, resp.Send)
	require.Equal(t, "голосование завершено - __\n- *за: 66% (2)*\n- *против: 33% (1) *", resp.Text)
	require.False(t, b.started)
}

func TestVotes_IgnoreStartVoteByProles(t *testing.T) {
	su := &MockSuperUser{}
	su.On("IsSuper", mock.Anything).Return(false)

	b := NewVotes(su)
	resp := b.OnMessage(Message{Text: "++"})

	require.False(t, resp.Send)
}

func TestVotes_IgnoreFinishVoteByProles(t *testing.T) {
	su := &MockSuperUser{}
	su.On("IsSuper", mock.Anything).Return(false)

	b := NewVotes(su)
	resp := b.OnMessage(Message{Text: "!!"})

	require.False(t, resp.Send)
}
