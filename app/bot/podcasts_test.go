package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodcasts_OnMessage(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			assert.Equal(t, "limit=5&q=Lambda", r.URL.RawQuery)
			sr := []siteAPIResp{
				{
					URL:       "http://example.com",
					Date:      time.Date(2020, 1, 31, 16, 45, 0, 0, time.UTC),
					ShowNum:   123,
					ShowNotes: "\n\n\nGo 2 начинается - 00:01:31.\nAWS Transfer for SFTP - 00:19:39.\nAWS App Mesh - 00:33:39.\nAmazon DynamoDB On-Demand - 00:46:50.\nALB сможет вызвать Lambda - 00:54:45.\nСлои общего кода в AWS Lambda - 01:15:46.\nDrone Cloud и бесплатно - 01:21:39.\nFoundationDB Document Layer совместим с mongo - 01:41:31.\nТемы наших слушателей\n\n\nСпонсор этого выпуска DigitalOcean\n\nаудио • лог чата\n\n",
				},
			}
			b, err := json.Marshal(sr)
			require.NoError(t, err)
			w.Write(b)
			return
		}
		w.WriteHeader(400)
	}))
	defer ts.Close()

	client := http.Client{Timeout: time.Second}
	d := NewPodcasts(&client, ts.URL, 5)

	result, answer := d.OnMessage(Message{Text: "/search Lambda"})
	require.True(t, answer)
	assert.Equal(t, "[Радио-Т #123](http://example.com) _31 Jan 20_\n- ALB сможет вызвать Lambda - 00:54:45.\n- Слои общего кода в AWS Lambda - 01:15:46.\n\n", result)

	_, answer = d.OnMessage(Message{Text: "/search Lambda"})
	require.True(t, answer, "second call ok too")
}

func TestPodcasts_OnMessageWithLinks(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			assert.Equal(t, "limit=5&q=mongo", r.URL.RawQuery)
			sr := []siteAPIResp{
				{
					URL:       "http://example.com",
					Date:      time.Date(2020, 1, 31, 16, 45, 0, 0, time.UTC),
					ShowNum:   123,
					ShowNotes: "\n\n\nMongo в облаке — чем это хорошо.\nWRT54GL Linksys живее всех.\nVSCode поддался.\nMozilla строит свой Context Graph.\nВоенное искуство для борьбы с хакерами.\nВдруг — Atom плагины.\nЛунный код открыт.\nxxxx в облаке — чем это хорошо\nТемы наших слушателей\n\n\nСпонсор этого выпуска DigitalOcean\n\nаудио • лог чата\n\n",
					Body:      "<p><img src=\"https://radio-t.com/images/radio-t/rt503.jpg\" alt=\"\" /></p>\n\n<ul>\n<li><a href=\"https://www.mongodb.com/cloud\">Mongo в облаке — чем это хорошо</a>.</li>\n<li><a href=\"http://arstechnica.com/information-technology/2016/07/the-wrt54gl-a-54mbps-router-from-2005-still-makes-millions-for-linksys/\">WRT54GL Linksys живее всех</a>.</li>\n<li><a href=\"https://code.visualstudio.com/updates\">VSCode поддался</a>.</li>\n<li><a href=\"http://venturebeat.com/2016/07/06/mozilla-is-building-context-graph-a-recommender-system-for-the-web/\">Mozilla строит свой Context Graph</a>.</li>\n<li><a href=\"http://www.businessinsider.com/cymettria-cyber-deception-2016-7\">Военное искуство для борьбы с хакерами</a>.</li>\n<li><a href=\"https://medium.com/@0x1AD2/atom-treasures-82a64ac391c\">Вдруг — Atom плагины</a>.</li>\n<li><a href=\"http://qz.com/726338/the-code-that-took-america-to-the-moon-was-just-published-to-github-and-its-like-a-1960s-time-capsule/\">Лунный код открыт</a>.</li>\n<li><a href=\"https://www.mongodb.com/cloud\">Mongo в облаке — чем это хорошо</a>.</li>\n<li>Темы наших слушателей</li>\n</ul>\n\n<p><em>Спонсор этого выпуска <a href=\"https://www.digitalocean.com\">DigitalOcean</a></em></p>\n\n<p><a href=\"https://cdn.radio-t.com/rt_podcast503.mp3\">аудио</a> • <a href=\"http://chat.radio-t.com/logs/radio-t-503.html\">лог чата</a>\n<audio src=\"https://cdn.radio-t.com/rt_podcast503.mp3\" preload=\"none\"></audio></p>\n",
				},
			}
			b, err := json.Marshal(sr)
			require.NoError(t, err)
			w.Write(b)
			return
		}
		w.WriteHeader(400)
	}))
	defer ts.Close()

	client := http.Client{Timeout: time.Second}
	d := NewPodcasts(&client, ts.URL, 5)

	result, answer := d.OnMessage(Message{Text: "/search mongo"})
	require.True(t, answer)
	assert.Equal(t, "[Радио-Т #123](http://example.com) _31 Jan 20_\n- [Mongo в облаке — чем это хорошо.](https://www.mongodb.com/cloud)\n- [xxxx в облаке — чем это хорошо](https://www.mongodb.com/cloud) 🔅\n\n", result)
}

func TestPodcasts_OnMessageIgnore(t *testing.T) {

	d := NewPodcasts(&http.Client{}, "http://example.com", 5)

	_, answer := d.OnMessage(Message{Text: "/xyz something"})
	require.False(t, answer)
}

func TestPodcasts_OnMessageFailed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer ts.Close()

	client := http.Client{Timeout: time.Second}
	d := NewPodcasts(&client, ts.URL, 5)

	_, answer := d.OnMessage(Message{Text: "/search something"})
	require.False(t, answer)
}

func TestPodcasts_notesWithLinks(t *testing.T) {
	s := siteAPIResp{
		URL:       "http://example.com",
		Date:      time.Date(2020, 1, 31, 16, 45, 0, 0, time.UTC),
		ShowNum:   123,
		ShowNotes: "\n\n\nMongo в облаке — чем это хорошо.\nWRT54GL Linksys живее всех.\nVSCode поддался.\nMozilla строит свой Context Graph.\nВоенное искуство для борьбы с хакерами.\nВдруг — Atom плагины.\nЛунный код открыт.\nТемы наших слушателей\n\n\nСпонсор этого выпуска DigitalOcean\n\nаудио • лог чата\n\n",
		Body:      "<p><img src=\"https://radio-t.com/images/radio-t/rt503.jpg\" alt=\"\" /></p>\n\n<ul>\n<li><a href=\"https://www.mongodb.com/cloud\">Mongo в облаке — чем это хорошо</a>.</li>\n<li><a href=\"http://arstechnica.com/information-technology/2016/07/the-wrt54gl-a-54mbps-router-from-2005-still-makes-millions-for-linksys/\">WRT54GL Linksys живее всех</a>.</li>\n<li><a href=\"https://code.visualstudio.com/updates\">VSCode поддался</a>.</li>\n<li><a href=\"http://venturebeat.com/2016/07/06/mozilla-is-building-context-graph-a-recommender-system-for-the-web/\">Mozilla строит свой Context Graph</a>.</li>\n<li><a href=\"http://www.businessinsider.com/cymettria-cyber-deception-2016-7\">Военное искуство для борьбы с хакерами</a>.</li>\n<li><a href=\"https://medium.com/@0x1AD2/atom-treasures-82a64ac391c\">Вдруг — Atom плагины</a>.</li>\n<li><a href=\"http://qz.com/726338/the-code-that-took-america-to-the-moon-was-just-published-to-github-and-its-like-a-1960s-time-capsule/\">Лунный код открыт</a>.</li>\n<li>Темы наших слушателей</li>\n</ul>\n\n<p><em>Спонсор этого выпуска <a href=\"https://www.digitalocean.com\">DigitalOcean</a></em></p>\n\n<p><a href=\"https://cdn.radio-t.com/rt_podcast503.mp3\">аудио</a> • <a href=\"http://chat.radio-t.com/logs/radio-t-503.html\">лог чата</a>\n<audio src=\"https://cdn.radio-t.com/rt_podcast503.mp3\" preload=\"none\"></audio></p>\n",
	}

	p := Podcasts{}
	r := p.notesWithLinks(s)

	exp := []noteWithLink{
		{text: "Mongo в облаке — чем это хорошо.", link: "https://www.mongodb.com/cloud"},
		{text: "WRT54GL Linksys живее всех.", link: "http://arstechnica.com/information-technology/2016/07/the-wrt54gl-a-54mbps-router-from-2005-still-makes-millions-for-linksys/"},
		{text: "VSCode поддался.", link: "https://code.visualstudio.com/updates"},
		{text: "Mozilla строит свой Context Graph.", link: "http://venturebeat.com/2016/07/06/mozilla-is-building-context-graph-a-recommender-system-for-the-web/"},
		{text: "Военное искуство для борьбы с хакерами.", link: "http://www.businessinsider.com/cymettria-cyber-deception-2016-7"},
		{text: "Вдруг — Atom плагины.", link: "https://medium.com/@0x1AD2/atom-treasures-82a64ac391c"},
		{text: "Лунный код открыт.", link: "http://qz.com/726338/the-code-that-took-america-to-the-moon-was-just-published-to-github-and-its-like-a-1960s-time-capsule/"},
	}
	assert.Equal(t, exp, r)
}

func TestPodcasts_notesWithLinks2(t *testing.T) {
	s := siteAPIResp{
		URL:       "http://example.com",
		Date:      time.Date(2020, 1, 31, 16, 45, 0, 0, time.UTC),
		ShowNum:   123,
		ShowNotes: "\n\n\nПочему квесты не помогают\nКак сделать резюме менее гадким\nSLB от Bobuk\nSLB от Umputun\nКакой длины строка еще работает\nПроблемы и решения mongo лока\nТемы наших слушателей\n\n\nаудио • radio-t.torrent • лог чата\n",
		Body:      "<p><img src=\"https://radio-t.com/images/radio-t/rt271.jpg\" alt=\"\" /></p>\n\n<ul>\n<li>Почему <a href=\"http://37signals.com/svn/posts/3071-why-we-dont-hire-programmers-based-on-puzzles-api-quizzes-math-riddles-or-other-parlor-trick\">квесты</a> не помогают</li>\n<li>Как сделать <a href=\"http://java.dzone.com/articles/how-make-your-cv-not-suck\">резюме</a> менее гадким</li>\n<li>SLB от Bobuk</li>\n<li>SLB от Umputun</li>\n<li>Какой длины строка еще работает</li>\n<li>Проблемы и решения mongo <a href=\"http://blog.pythonisito.com/2011/12/mongodbs-write-lock.html\">лока</a></li>\n<li>Темы наших слушателей</li>\n</ul>\n\n<p><a href=\"https://cdn.radio-t.com/rt_podcast271.mp3\">аудио</a> • <a href=\"https://cdn.radio-t.com/torrents/rt_podcast271.mp3.torrent\">radio-t.torrent</a> • <a href=\"http://chat.radio-t.com/logs/radio-t-271.html\">лог чата</a><audio src=\"https://cdn.radio-t.com/rt_podcast271.mp3\" preload=\"none\"></audio></p>\n",
	}

	p := Podcasts{}
	r := p.notesWithLinks(s)

	exp := []noteWithLink{
		{text: "Почему квесты не помогают", link: "http://37signals.com/svn/posts/3071-why-we-dont-hire-programmers-based-on-puzzles-api-quizzes-math-riddles-or-other-parlor-trick"},
		{text: "Как сделать резюме менее гадким", link: "http://java.dzone.com/articles/how-make-your-cv-not-suck"},
		{text: "SLB от Bobuk", link: ""},
		{text: "SLB от Umputun", link: ""},
		{text: "Какой длины строка еще работает", link: ""},
		{text: "Проблемы и решения mongo лока", link: "http://blog.pythonisito.com/2011/12/mongodbs-write-lock.html"},
	}
	assert.Equal(t, exp, r)
}
