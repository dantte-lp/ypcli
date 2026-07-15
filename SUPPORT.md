# Support

## Support Channels

| Request type | Channel | Required information |
|---|---|---|
| Bug report | GitHub issue form | ypcli version (`ypcli version`), OS/arch, exact command, `--verbose` output, expected vs actual |
| Feature request | GitHub issue form | Use case, expected behavior, affected command |
| Documentation issue | GitHub issue form | Affected page, expected correction |
| Security vulnerability | GitHub Security Advisory | Affected versions, impact, reproduction, mitigation |

## Unsupported Channels

- Security vulnerabilities are not handled in public issues.
- Private support SLAs are not provided by this repository.

## Compatibility Scope

| Area | Policy |
|---|---|
| Go version | See `go.mod` and CI configuration. |
| Operating systems | macOS, Linux, Windows (amd64 + arm64). |
| yopass server | Interoperable with yopass v13+ API; older servers work minus `/version` and token auth. |
| Public API | Pre-1.0 stability follows Semantic Versioning rule 4. |
| Releases | Supported version is the latest published release unless stated otherwise in `SECURITY.md`. |

## References

- Security policy: [SECURITY.md](./SECURITY.md)
- Contributing rules: [CONTRIBUTING.md](./CONTRIBUTING.md)
- Development workflow: [docs/en/08-development.md](./docs/en/08-development.md)
