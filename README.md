# Telegram бот подкаста Радио-Т

[![Build Status](https://github.com/radio-t/gitter-rt-bot/workflows/build/badge.svg)](https://github.com/radio-t/gitter-rt-bot/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/radio-t/gitter-rt-bot)](https://goreportcard.com/report/github.com/radio-t/gitter-rt-bot) [![Coverage Status](https://coveralls.io/repos/github/radio-t/super-bot/badge.svg?branch=master)](https://coveralls.io/github/radio-t/super-bot?branch=master)

## Основная функциональность

HanucaHo Ha PacTe

Бот слушает [чат Telegram](https://t.me/radio_t_chat) и реагирует на определенные команды и фрагменты текста.
Кроме этого, он слушает API новостей и публикует в Telegram сообщения о начале выпуска и смене тем.

С ботом можно [общаться тет-а-тет](https://t.me/radiot_superbot), не засорая общий чат.

В режиме экспортирования сохраняет лог сообщений в HTML файл.

## Статус

Бот в работе несколько лет и успешно "участвовал" во многих подкастах. 

## Команды бота

| Команда        | Описание                                                                                                       |
|----------------|----------------------------------------------------------------------------------------------------------------|
| `ping`, `пинг` | ответит `pong`, `понг`, см. [basic.data](https://github.com/radio-t/gitter-rt-bot/blob/master/data/basic.data) |
| `анекдот!`, `анкедот!`, `joke!`, `chuck!` | расскажет анекдот с rzhunemogu.ru или icndb.com (нужен `MASHAPE_TOKEN`)             |
| `news!`, `новости!`                       | 5 последних [новостей для Радио-Т](https://news.radio-t.com)                        |
| `so!`                                     | 1 вопрос со [Stackoverflow](https://stackoverflow.com/questions?tab=Active)         |
| `?? <запрос>`, `/ddg <запрос>`                             | поискать "<запрос>" на [DuckDuckGo](https://duckduckgo.com)                         |
| `search! <слово>`, `/search <слово>` | поискать по шоунотам подкастов|

## Инструкции по локальной разработке

Для создания тестового бота нужно обратиться к [BotFather](https://t.me/BotFather) и получить от него токен.

После создания бота нужно вручную добавить в группу (Info / Add Members) и дать права администратора (Info / Edit / Administrators / Add Admin).

Приложение ожидает следующие переменные окружения:

* `TELEGRAM_TOKEN` – токен полученный от BotFather
* `TELEGRAM_GROUP` - основная группа в Телеграмме (туда приходят уведомления о новостях, все сообщения сохраняются в лог)
* `MASHAPE_TOKEN` – токен от сервиса [Kong](https://konghq.com/), используется только для DuckDuckGo бота

Дополнительные переменные окружения со значениями по-умолчанию:

* `DEBUG` (false) – включает режим отладки (логируется больше событий)
* `TELEGRAM_LOGS` (logs) - путь к папке куда пишется лог чата
* `SYS_DATA` (data) - путь к папке с *.data файлами и шаблоном для построения HTML отчета
* `TELEGRAM_TIMEOUT` (30s) – HTTP таймаут для скачивания файлов из Telegram при построении HTML отчета
* `RTJC_PORT` (18001) – порт на который приходят уведомления о новостях

Запустить бота можно через Docker Compose:

```bash
docker-compose up telegram-bot
```

Или с помощью Make:

```bash
make run ARGS="--super=umputun --super=bobuk --super=grayru --super=ksenks"
```

Для построения HTML отчета необходимо передать дополнительные флаги:

```bash
docker-compose exec telegram-bot ./telegram-rt-bot \
  --super=umputun \
  --super=bobuk \
  --super=grayru \
  --super=ksenks \
  --export-num=688 \
  --export-path=html \
  --export-day=20200208 \
  --export-template=logs.html
```

или

```bash
make run ARGS="--super=umputun --super=bobuk --super=grayru --super=ksenks --export-num=688 --export-path=logs --export-day=20200208 --export-template=data/logs.html"
```
