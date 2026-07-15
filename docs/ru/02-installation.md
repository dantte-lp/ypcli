# Установка

ypcli поставляется в виде статического бинарного файла без CGO для macOS, Linux и Windows
на архитектурах amd64 и arm64.

## Homebrew (macOS)

```bash
brew install dantte-lp/tap/ypcli
```

## Scoop (Windows)

```powershell
scoop bucket add dantte-lp https://github.com/dantte-lp/scoop-bucket
scoop install ypcli
```

## winget (Windows)

```powershell
winget install dantte-lp.ypcli
```

## Go

```bash
go install github.com/dantte-lp/ypcli/cmd/ypcli@latest
```

## Готовые бинарные файлы

Скачайте архив для вашей платформы со страницы
[Releases](https://github.com/dantte-lp/ypcli/releases), проверьте его по
`checksums.txt`, распакуйте и поместите `ypcli` в ваш `PATH`.

```bash
tar -xzf ypcli_*_linux_amd64.tar.gz
sudo install ypcli /usr/local/bin/
ypcli version
```

## Автодополнение в оболочке

ypcli генерирует скрипты автодополнения для bash, zsh, fish и powershell:

```bash
# zsh
ypcli completion zsh > "${fpath[1]}/_ypcli"

# bash
ypcli completion bash | sudo tee /etc/bash_completion.d/ypcli >/dev/null
```

## Проверка сборки

```bash
ypcli version --json
```

Вывод содержит версию клиента, коммит, дату сборки и версию целевого
сервера.
