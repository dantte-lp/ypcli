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
