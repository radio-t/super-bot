package events

import (
	"strings"

	"github.com/sromku/go-gitter"
)

// SuperUser for moderators
type SuperUser []string

// IsSuper checks if gitter user in su list
func (s SuperUser) IsSuper(user gitter.User) bool {
	for _, super := range s {
		if strings.EqualFold(user.Username, super) || strings.EqualFold("/"+user.Username, super) {
			return true
		}
	}
	return false
}
