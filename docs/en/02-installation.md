# Installation

ypcli ships as a static, CGO-free binary for macOS, Linux, and Windows on both
amd64 and arm64.

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

## Prebuilt binaries

Download an archive for your platform from the
[Releases](https://github.com/dantte-lp/ypcli/releases) page, verify it against
`checksums.txt`, extract, and place `ypcli` on your `PATH`.

```bash
tar -xzf ypcli_*_linux_amd64.tar.gz
sudo install ypcli /usr/local/bin/
ypcli version
```

## Shell completion

ypcli generates completion scripts for bash, zsh, fish, and powershell:

```bash
# zsh
ypcli completion zsh > "${fpath[1]}/_ypcli"

# bash
ypcli completion bash | sudo tee /etc/bash_completion.d/ypcli >/dev/null
```

## Verifying the build

```bash
ypcli version --json
```

The output includes the client version, commit, build date, and the target
server's version.
