# Использование

## Отправка секретов

Текст из stdin:

```bash
printf 'my secret' | ypcli send
```

Текст из флага:

```bash
ypcli send --text 'my secret'
```

Файл (публикуется как файловый секрет; имя файла встраивается в зашифрованную
полезную нагрузку, никогда не в открытом виде):

```bash
ypcli send --file ./db.env --expiration 1d
```

### Параметры

```bash
# Multi-view secret, valid for a week
cat notes.md | ypcli send --expiration 1w --one-time=false

# Show a scannable QR code and copy the URL to the clipboard
printf 'wifi-password' | ypcli send --qr --copy

# Manual key: the key is omitted from the URL and printed separately,
# so you can deliver it out of band
ypcli send --file secret.pem --key "$(openssl rand -hex 16)"
```

### Составление в редакторе

Запустите `send` интерактивно (без `--file`/`--text`/stdin), и ypcli откроет ваш
редактор; секрет отправляется после сохранения и выхода. Редактор берётся из
`$YPCLI_EDITOR`, `$VISUAL` или `$EDITOR` (с fallback на `vi`, либо `notepad` на
Windows). `--editor` принудительно включает режим даже при конвейерном вводе.

```bash
ypcli send                 # открывает редактор
ypcli send --editor --expiration 1d
```

### Из менеджера секретов (Vault / OpenBao)

Читайте payload прямо из движка Vault или OpenBao KV v2 — ничего не попадает в
историю shell или на файловую систему. Учитываются стандартные переменные
окружения `VAULT_*` / `BAO_*`.

```bash
export VAULT_ADDR=https://vault.corp VAULT_TOKEN=…
ypcli send --vault-path db --vault-field password
```

### Из любого менеджера секретов

`--input-command` выполняет любую команду и отправляет её **сырой stdout** как
секрет — универсальный мост к любому инструменту (AWS Secrets Manager, gopass,
`pass`, 1Password CLI, …). Большинство менеджеров добавляют завершающий перевод
строки; уберите его в команде, если нужно точное значение.

```bash
ypcli send --input-command 'pass show db/password'
ypcli send --input-command 'op read op://vault/db/password'
ypcli send --input-command 'aws secretsmanager get-secret-value --secret-id db --query SecretString --output text'
```

Для bearer-токена, которым аутентифицируются к приватному *yopass*-серверу,
используйте `token_command` — см. [Конфигурация](05-configuration.md#токены).

## Получение секретов

По ссылке для обмена:

```bash
ypcli receive 'https://yopass.se/#/s/ID/KEY'
```

Ссылка с ручным ключом требует `--key`:

```bash
ypcli receive 'https://yopass.se/#/c/ID' --key MANUALKEY
```

Без URL используйте `--id`/`--key` (добавьте `--file` для файловых секретов):

```bash
ypcli receive --id ID --key KEY --file -o ./downloads/
```

### Вывод файлов

- Без `--output` файловый секрет записывается под встроенным именем файла в
  текущий каталог.
- `--output DIR/` (каталог, существующий или с завершающим разделителем) записывает
  файл под встроенным именем, создавая каталог при необходимости.
- `--output PATH` записывает по этому точному пути.

## Глобальные флаги

Все команды принимают глобальные флаги, описанные в
[Справочнике CLI](04-cli.md) и [Конфигурации](05-configuration.md), включая
`--profile`, `--api`, `--url`, `--token`, `--timeout`, `--json` и
`--verbose`.
