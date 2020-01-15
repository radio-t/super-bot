package events

import (
	"context"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func TestSendStartMessageOnBroadcastStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	submitter := NewMockSubmitter(ctrl)
	submitter.EXPECT().Submit(gomock.Any(), MsgBroadcastStarted)

	bot := NewBroadcastStatusBot(time.Millisecond, "", 3*time.Millisecond, "", submitter)
	bot.pingFn = func(ctx context.Context) bool {
		return true
	}

	bot.process(context.Background())
}

func TestSendStopMessageOnBroadcastStopAndIntervalDeadline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	submitter := NewMockSubmitter(ctrl)
	submitter.EXPECT().Submit(gomock.Any(), MsgBroadcastFinished)

	bot := NewBroadcastStatusBot(time.Second/10, "", 3*time.Second/10, "", submitter)
	bot.status = true
	bot.lastOnStateTime = time.Now()
	bot.pingFn = func(ctx context.Context) bool {
		return false
	}

	bot.process(context.Background())
	bot.process(context.Background())
	time.Sleep(4 * time.Second / 10)
	bot.process(context.Background())
}

func TestDoNothingIfStatusNotChanged(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	submitter := NewMockSubmitter(ctrl)
	bot := NewBroadcastStatusBot(time.Millisecond, "", 3*time.Millisecond, "", submitter)
	bot.pingFn = func(ctx context.Context) bool {
		return false
	}

	bot.process(context.Background())
}
