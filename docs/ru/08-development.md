# Разработка

## Требования

- Go (см. `go.mod` для требуемой версии)
- `golangci-lint` v2
- Для end-to-end тестов: [`uv`] и движок контейнеров (`podman` или `docker`)
- Опционально: `goreleaser`, `markdownlint-cli2`, `yamllint`, `cspell`

## Цели Make

```bash
make build       # build the ypcli binary with version ldflags
make test        # go test -race -cover ./...
make lint        # golangci-lint v2
make lint-docs   # markdownlint + yamllint + cspell
make vuln        # govulncheck ./...
make verify      # build + test + lint + vuln
make e2e         # end-to-end suite (uv + ruff + ty + live yopass container)
make snapshot    # local goreleaser snapshot
```

## Структура

```text
cmd/ypcli/        entrypoint (version ldflags)
internal/
  cli/            cobra commands, resolution, exit codes
  api/            HTTP transport
  crypto/         vendored OpenPGP (interop-critical)
  config/         profiles, token sourcing
  output/         printers, qr, progress
  clipboard/      clipboard wrapper
tests/e2e/        Python end-to-end suite (uv + ruff + ty)
docs/en, docs/ru  bilingual documentation
```

## Барьер совместимости

`internal/crypto/interop_test.go` импортирует
`github.com/jhaals/yopass/pkg/yopass` как зависимость **только для тестов** и
проверяет двусторонние round-trip'ы (ypcli ↔ upstream, текст + файл + Argon2).
Он должен проходить при любом изменении криптографии.

Убедитесь, что зависимость никогда не линкуется в бинарник:

```bash
go build -o /tmp/ypcli ./cmd/ypcli
go tool nm /tmp/ypcli | grep -c jhaals/yopass   # expect 0
```

## End-to-end тесты

Go-юнит-тесты покрывают функции по отдельности; набор `tests/e2e/` — это слой
чёрного ящика, который гоняет **собранный бинарник `ypcli`** против **живого
сервера yopass** (запускаемого в контейнере) и проверяет каждую команду, флаг и
код возврата от начала до конца. Это Python-проект под управлением [`uv`],
линтуемый [`ruff`] и проверяемый по типам [`ty`].

```bash
make e2e
# либо из tests/e2e:
uv run ruff check .
uv run ty check .
uv run pytest -v
```

Session-фикстура собирает бинарник один раз и запускает `memcached` +
`jhaals/yopass` через `podman`/`docker`; небольшой in-process фейковый сервер
покрывает случаи, которые бесплатный образ не воспроизводит детерминированно
(аутентификация `401`, отсутствие `/version`, перехват заголовков запроса).
Криптографическая совместимость с настоящим yopass/openpgp.js доказывается
отдельно [барьером совместимости](#барьер-совместимости).

Полезные переменные окружения:

| Переменная | Эффект |
|---|---|
| `YPCLI_BIN` | Использовать существующий бинарник `ypcli` вместо сборки |
| `YPCLI_E2E_API` | Указать на уже запущенный сервер yopass (пропустить старт контейнера) |
| `YPCLI_E2E_ARGON2` | Установите `1`, если у внешнего сервера включён Argon2 |

Покрытие описано в [`tests/e2e/README.md`](../../tests/e2e/README.md). Workflow
`e2e` в GitHub Actions прогоняет набор при каждом push и pull request.

## Стандарты написания кода

- Оборачивайте ошибки через `%w`; классифицируйте через `errors.Is`/`errors.As`.
- `context.Context` — первый параметр, никогда не хранится в структуре.
- Логирование только через `log/slog`.
- Табличные тесты, `t.Parallel()` где это безопасно, всегда `-race`.
- Conventional Commits для сообщений; см. [CONTRIBUTING.md](../../CONTRIBUTING.md).

## Выпуск релизов

Релизы создаются путём отправки SemVer-тега; goreleaser собирает матрицу
платформ и публикует архивы, а также манифесты Homebrew/Scoop/winget.

```bash
make verify
git tag -a v0.1.0 -m "ypcli v0.1.0"
git push origin v0.1.0
```

Рабочий процесс релиза требует `TAP_GITHUB_TOKEN` (доступ на запись к
репозиториям tap, bucket и winget) в дополнение к стандартному `GITHUB_TOKEN`.

[`uv`]: https://docs.astral.sh/uv/
[`ruff`]: https://docs.astral.sh/ruff/
[`ty`]: https://github.com/astral-sh/ty
