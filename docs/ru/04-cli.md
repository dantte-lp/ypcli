# Справочник CLI

## Глобальные флаги

Доступны для каждой команды. Приоритет разрешения:
**флаг > env (`YPCLI_*`) > активный профиль > глобальные дефолты > встроенное значение по умолчанию**.

| Флаг | Env | Описание |
|---|---|---|
| `--profile, -p` | `YPCLI_PROFILE` | используемый профиль конфигурации |
| `--api` | `YPCLI_API` | базовый URL API yopass |
| `--url` | `YPCLI_URL` | публичный URL yopass (для ссылок обмена) |
| `--token` | `YPCLI_TOKEN` | bearer-токен для инстансов с аутентификацией |
| `--timeout` | `YPCLI_TIMEOUT` | тайм-аут запроса (по умолчанию `30s`) |
| `--json` | `YPCLI_JSON` | машиночитаемый вывод в формате JSON |
| `--verbose, -v` | `YPCLI_VERBOSE` | отладочное логирование в stderr |
| `--config` | `YPCLI_CONFIG` | путь к файлу конфигурации |

## `ypcli send`

Зашифровать и опубликовать секрет. Ввод поступает из `--vault-path`, `--file`, `--text`, переданного через конвейер stdin или из редактора (при интерактивном запуске).

| Флаг | Описание |
|---|---|
| `--file, -f` | прочитать секрет из файла (публикуется как файловый секрет) |
| `--text, -t` | текст секрета (вместо stdin/файла) |
| `--input-command` | выполнить команду и использовать её stdout как секрет |
| `--editor` | составить секрет в `$EDITOR` (по умолчанию при интерактивном запуске) |
| `--expiration, -e` | время жизни: `1h`, `1d` или `1w` (по умолчанию `1h`) |
| `--one-time` | удалить после первого просмотра (по умолчанию `true`) |
| `--require-auth` | требовать аутентификацию для просмотра (нужна поддержка на сервере) |
| `--key, -k` | ручной ключ шифрования; исключается из URL |
| `--qr` | также вывести URL в виде терминального QR-кода (текстовый режим) |
| `--copy` | скопировать URL в системный буфер обмена |
| `--vault-path` | взять payload из Vault/OpenBao KV v2 по указанному пути |
| `--vault-field` | поле, которое читать из секрета Vault/OpenBao |
| `--vault-mount` | mount KV v2 (по умолчанию `secret`) |
| `--vault-addr` | адрес Vault/OpenBao (по умолчанию `$VAULT_ADDR` / `$BAO_ADDR`) |
| `--vault-token` | токен Vault/OpenBao (по умолчанию `$VAULT_TOKEN` / `$BAO_TOKEN`) |
| `--vault-namespace` | namespace (по умолчанию `$VAULT_NAMESPACE` / `$BAO_NAMESPACE`) |

Приоритет ввода: `--vault-path` > `--input-command` > `--file` > `--text` >
конвейерный stdin > редактор.

Вывод JSON:

```json
{"id":"…","url":"https://…/#/s/…/…","key":"…","manual_key":false,"file":false,"one_time":true,"expiration":"1h"}
```

## `ypcli receive`

Получить и расшифровать секрет. Принимает позиционный аргумент — ссылку для обмена, либо
`--id`/`--key`.

| Флаг | Описание |
|---|---|
| `--id` | ID секрета (если URL не указан) |
| `--key, -k` | ключ дешифрования (обязателен для ссылок с ручным ключом и `--id`) |
| `--file` | обрабатывать секрет как файл (вместе с `--id`) |
| `--output, -o` | выходной файл или каталог для файловых секретов |

- Текстовые секреты записываются в **stdout**.
- Файловые секреты записываются под их исходным именем или в путь, заданный `-o`.

## `ypcli config`

Управление именованными профилями серверов в `$XDG_CONFIG_HOME/ypcli/config.yaml` (режим 0600).

```bash
ypcli config add work --api https://api.corp --url https://yp.corp \
  --expiration 1d --token-command 'vault read -field=token secret/yopass'
ypcli config list      # * marks the active profile
ypcli config use work
ypcli config remove work
```

| Подкоманда | Флаги |
|---|---|
| `add <name>` | `--api`, `--url`, `--expiration`, `--token-command`, `--vault-addr`, `--vault-mount`, `--vault-namespace`, `--vault-token-command` |
| `list` | — |
| `use <name>` | — |
| `remove <name>` | — |
| `defaults` | как у `add`, сохраняется в глобальные дефолты |

## `ypcli version`

Выводит информацию о сборке клиента (версия/коммит/дата) и запрашивает у сервера эндпоинт
`/version`. Серверы старше yopass 13.x возвращают `unsupported`.

```bash
ypcli version --api https://api.yopass.se --json
```

## `ypcli mcp`

Запустить MCP-сервер, экспонирующий операции send/receive ypcli ИИ-агентам. См.
[MCP-сервер](09-mcp.md) для полного руководства.

| Флаг | Описание |
|---|---|
| `--http` | обслуживать по HTTP на этом адресе вместо stdio (напр. `127.0.0.1:8765`) |
| `--http-token` | bearer-токен, обязательный в HTTP-режиме (`$YPCLI_MCP_TOKEN`) |
| `--read-only` | экспонировать только send-инструменты (без `receive_secret`) |

```bash
ypcli mcp                                   # stdio (для локального агента)
YPCLI_MCP_TOKEN=… ypcli mcp --http :8765    # общий HTTP-сервер
```

## `ypcli completion`

Сгенерировать скрипт автодополнения оболочки для `bash`, `zsh`, `fish` или `powershell`.

```bash
ypcli completion zsh > "${fpath[1]}/_ypcli"
```

## Коды возврата

| Код | Значение |
|---|---|
| 0 | успех |
| 1 | общая ошибка |
| 2 | использование / неверные флаги |
| 3 | ошибка конфигурации |
| 4 | сеть / тайм-аут |
| 5 | сбой аутентификации (401/403) |
| 6 | не найдено / one-time уже использован (404/410) |
| 7 | сбой дешифрования / криптографии |
