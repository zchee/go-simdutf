# go-simdutf

[![test][test-badge]][test]
[![pkg.go.dev][pkg.go.dev-badge]][pkg.go.dev]
[![Go module][module-badge]][module]
[![codecov.io][codecov-badge]][codecov]


Package simdutf provides a Go port of [simdutf/simdutf](https://github.com/simdutf/simdutf).

Unicode routines (UTF8, UTF16, UTF32) and Base64: billions of characters per second using SSE2, AVX2, NEON, AVX-512, RISC-V Vector Extension, LoongArch64, POWER.

<!-- badge links -->
[test]: https://github.com/zchee/go-simdutf/actions/workflows/ci.yaml
[pkg.go.dev]: https://pkg.go.dev/github.com/zchee/go-simdutf
[module]: https://github.com/zchee/go-simdutf/releases/latest
[codecov]: https://app.codecov.io/gh/zchee/go-simdutf

[test-badge]: https://img.shields.io/github/actions/workflow/status/zchee/go-simdutf/ci.yaml?branch=main&style=for-the-badge&label=TEST&logo=github
[pkg.go.dev-badge]: https://img.shields.io/badge/pkg.go.dev-doc-00add8?style=for-the-badge&logo=go
[module-badge]: https://img.shields.io/github/release/zchee/go-simdutf.svg?color=00add8&label=MODULE&style=for-the-badge&logo=go
[codecov-badge]: https://img.shields.io/codecov/c/zchee/go-simdutf/main?logo=codecov&style=for-the-badge
