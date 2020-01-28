# Telegram бот подкаста Радио-Т

[![Build Status](https://github.com/radio-t/gitter-rt-bot/workflows/build/badge.svg)](https://github.com/radio-t/gitter-rt-bot/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/radio-t/gitter-rt-bot)](https://goreportcard.com/report/github.com/radio-t/gitter-rt-bot) [![Coverage Status](https://coveralls.io/repos/github/radio-t/super-bot/badge.svg?branch=master)](https://coveralls.io/github/radio-t/super-bot?branch=master)

## Основная функциональность

Бот слушает чат telegram и реагирует на определенные команды и фрагменты текста. Кроме этого, он слушает
API новостей и публикует в telegram сообщения о начале выпуска и смене тем.

В режиме экспортирования сохраняет лог сообщений в html файл.

## Статус

Бот в работе несколько лет и успешно "участвовал" во многих подкастах. 

## Команды бота

| Команда        | Описание                                                                                                       |
|----------------|----------------------------------------------------------------------------------------------------------------|
| `ping`, `пинг` | ответит `pong`, `понг`, см. [basic.data](https://github.com/radio-t/gitter-rt-bot/blob/master/data/basic.data) |
| `анекдот!`, `анкедот!`, `joke!`, `chuck!` | расскажет анекдот с rzhunemogu.ru или icndb.com (нужен `MASHAPE_TOKEN`)             |
| `++`                                      | начать голосование (только для супер-пользователей)                                 |
| `+1`                                      | проголосовать "за"                                                                  |
| `-1`                                      | проголосовать "против"                                                              |
| `!!`                                      | закончить голосование (только для супер-пользователей)                              |
| `news!`, `новости!`                       | 5 последних [новостей для Радио-Т](https://news.radio-t.com)                        |
| `so!`                                     | 1 вопрос со [Stackoverflow](https://stackoverflow.com/questions?tab=Active)         |
| `?? <запрос>`                             | поискать "<запрос>" на [DuckDuckGo](https://duckduckgo.com)                         |
