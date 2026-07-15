# Разработка

## Требования

- Go (см. `go.mod` для требуемой версии)
- `golangci-lint` v2
- Опционально: `goreleaser`, `markdownlint-cli2`, `yamllint`, `cspell`

## Цели Make

```bash
make build       # build the ypcli binary with version ldflags
make test        # go test -race -cover ./...
make lint        # golangci-lint v2
make lint-docs   # markdownlint + yamllint + cspell
make vuln        # govulncheck ./...
make verify      # build + test + lint + vuln
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
