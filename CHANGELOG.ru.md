# Changelog

Все значимые изменения этого проекта документируются в этом файле.

Формат основан на [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/),
и этот проект придерживается [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).

English version: [CHANGELOG.md](CHANGELOG.md).

## [Unreleased]

### Добавлено

- **`ypcli mcp`** — сервер Model Context Protocol, экспонирующий send/receive
  ИИ-агентам (Claude, Codex, Gemini). Инструменты: `send_secret`, `send_file`,
  `receive_secret` (убирается через `--read-only`), `list_profiles`,
  `server_version`. Работает по stdio или HTTP (`--http`, защита bearer-токеном).
  Поставляет Claude Agent Skill (`skills/ypcli/`), конфиги для клиентов
  (`integrations/`) и hardened systemd-юнит (`deploy/`).
- `ypcli send --input-command '<cmd>'` выполняет любую команду и отправляет её
  сырой stdout как секрет — универсальный мост к любому менеджеру секретов
  (AWS Secrets Manager, gopass, `pass`, 1Password CLI, …).
- Чтение payload секрета напрямую из движка HashiCorp Vault или OpenBao KV v2:
  `ypcli send --vault-path <path> --vault-field <field>` (с
  `--vault-addr/-token/-mount/-namespace`, учитывая env `VAULT_*` / `BAO_*`).
  Настройки подключения также могут жить в блоке `vault` профиля (задаются через
  `ypcli config add/defaults --vault-…`); приоритет: флаг > env > профиль.
- Режим редактора: `ypcli send` без ввода открывает `$EDITOR` (или `--editor`)
  для составления секрета и отправляет его по сохранению.
- Глобальные дефолты: блок `defaults` верхнего уровня в файле конфигурации
  (управляется через `ypcli config defaults`), применяемый под каждым профилем,
  чтобы указать self-hosted сервер без создания профиля. Приоритет теперь:
  флаг > env > активный профиль > глобальные дефолты > встроенный дефолт.
- Автоматизированный набор end-to-end тестов на Python (uv + ruff + ty),
  прогоняющий бинарник `ypcli` против живого контейнера yopass и покрывающий
  каждую команду, флаг и код возврата, плюс управляемый фейковый сервер для
  проверки аутентификации и кодов ошибок. Запуск через `make e2e` и CI-workflow
  `e2e`.

## [0.1.0] - 2026-07-15

### Добавлено

- **Команда `send`** — шифрует текст (stdin/`--text`) или файлы (`--file`) с
  помощью OpenPGP на стороне клиента и публикует one-time URL для секрета.
  Поддерживает `--expiration` (`1h`/`1d`/`1w`), `--one-time`, `--require-auth`,
  ручной `--key`, `--qr` (QR-код в терминале) и `--copy` (системный буфер обмена).
- **Команда `receive`** — получает и расшифровывает секрет по URL или
  `--id`/`--key`. Текст записывается в stdout; файлы записываются под встроенным
  именем файла или в `--output` (с индикатором прогресса потоковой загрузки).
- **Команда `config`** — управляет именованными профилями серверов
  (`add`/`list`/`use`/`remove`) в `$XDG_CONFIG_HOME/ypcli/config.yaml` (права 0600).
- **Команда `version`** — сообщает о сборке клиента и о серверном эндпоинте
  `/version`, корректно деградируя на серверах до версии 13.x.
- **Аутентификация по bearer token** — `--token`, `YPCLI_TOKEN` или
  `token_command` конкретного профиля; токены никогда не сохраняются на диск.
- **Машиночитаемый вывод** — `--json` в каждой команде, а также стабильные коды
  возврата (2 использование, 3 конфигурация, 4 сеть, 5 аутентификация,
  6 не найдено/использовано, 7 криптография).
- **Автоопределение Argon2id** — выведение ключа выбирается для каждого запроса
  по серверному эндпоинту `/config`.
- **Кроссплатформенный релиз** — матрица goreleaser для macOS/Linux/Windows на
  amd64 + arm64, с публикацией Homebrew cask, Scoop и winget.

### Исправлено

- Строка прогресса скачивания файла теперь завершается переводом строки после
  окончания потока, даже если reader сообщает `io.EOF` отдельным нулевым
  чтением (например, HTTP-тела).

### Безопасность

- Побайтовая совместимость OpenPGP с веб-фронтендом yopass (openpgp.js v6),
  доказанная round-trip тестом только для тестов против вышестоящего yopass,
  который никогда не линкуется в поставляемый бинарник.

[Unreleased]: https://github.com/dantte-lp/ypcli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/dantte-lp/ypcli/releases/tag/v0.1.0
