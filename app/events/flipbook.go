package events

import (
	"fmt"
	"time"

	"github.com/go-pkgz/lcw"

	"github.com/radio-t/super-bot/app/bot"
)

// Flipbook is a storage for multi-text messages
type Flipbook struct {
	// Cache is using to keep in memory last multi-text messages
	cache   lcw.LoadingCache
	counter int
}

// NewFlipbook creates new flipbook object
func NewFlipbook() Flipbook {
	cache, _ := lcw.NewExpirableCache(lcw.MaxKeys(100), lcw.TTL(time.Hour*12))
	return Flipbook{
		cache:   cache,
		counter: 0,
	}
}

// Save message with multi-text to cache
func (f Flipbook) Save(message bot.Response) (key string, err error) {
	f.counter++
	key = fmt.Sprintf("%d", f.counter)
	_, err = f.cache.Get(key, func() (interface{}, error) {
		return message, nil
	})

	if err != nil {
		return "", err
	}

	return key, nil
}

func (f Flipbook) Get(key string, page int) (message string, prevPage, nextPage int, err error) {
	val, err := f.cache.Get(key, func() (interface{}, error) {
		return nil, fmt.Errorf("not found")
	})

	if err != nil {
		return "", -1, -1, err
	}

	msg := val.(bot.Response)
	if page == 0 {
		prevPage = -1
		if len(msg.AltText) > 0 {
			nextPage = 1
		} else {
			nextPage = -1
		}

		return msg.Text, prevPage, nextPage, nil
	}

	if page > len(msg.AltText) {
		return "", -1, -1, fmt.Errorf("page %d not found", page)
	}

	message = msg.AltText[page-1]

	if page > 0 {
		prevPage = page - 1
	} else {
		prevPage = -1
	}

	if page < len(msg.AltText) {
		nextPage = page + 1
	} else {
		nextPage = -1
	}

	return message, prevPage, nextPage, nil
}
