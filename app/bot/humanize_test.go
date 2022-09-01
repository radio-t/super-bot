package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHumanizeDuration(t *testing.T) {
	require.Equal(t, "2дн 3ч 4мин 5сек", HumanizeDuration(2*Day+3*time.Hour+4*time.Minute+5*time.Second))
	require.Equal(t, "600дн 3ч 4мин", HumanizeDuration(600*Day+3*time.Hour+4*time.Minute))
	require.Equal(t, "2дн 3ч", HumanizeDuration(2*Day+3*time.Hour))
	require.Equal(t, "2дн 4мин", HumanizeDuration(2*Day+4*time.Minute))
	require.Equal(t, "2дн", HumanizeDuration(2*Day))
	require.Equal(t, "3ч", HumanizeDuration(3*time.Hour))
	require.Equal(t, "4мин", HumanizeDuration(4*time.Minute))
}
