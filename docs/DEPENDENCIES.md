# 依存一覧

アプリは Go 1.27.1 を使う。
`go.mod` と `go.sum` は依存バージョンとチェックサムを固定する。
この表は `go list -m -json all` の全モジュールを記録する。
実行時の分類は `go list -deps -json ./cmd/mezzanine` を使う。
ライセンス名は配布元の原文を参照する。
配布者は `THIRD_PARTY_NOTICES.txt` をバイナリと配布する。

| モジュール | バージョン | ライセンス | 用途 |
| --- | --- | --- | --- |
| `github.com/PuerkitoBio/goquery` | `v1.8.0` | BSD-3-Clause | 間接・開発用 |
| `github.com/andybalholm/cascadia` | `v1.3.1` | BSD-2-Clause | 間接・開発用 |
| `github.com/doyensec/safeurl` | `v0.2.5` | Apache-2.0 | 実行時 |
| `github.com/dustin/go-humanize` | `v1.0.1` | MIT | 実行時 |
| `github.com/google/go-cmp` | `v0.6.0` | BSD-3-Clause | 間接・開発用 |
| `github.com/google/pprof` | `v0.0.0-20250317173921-a4b03ec1a45e` | Apache-2.0 | 間接・開発用 |
| `github.com/google/uuid` | `v1.6.0` | BSD-3-Clause | 実行時 |
| `github.com/gorilla/feeds` | `v1.2.0` | BSD-3-Clause | 実行時 |
| `github.com/hashicorp/golang-lru/v2` | `v2.0.7` | MPL-2.0 | 間接・開発用 |
| `github.com/kr/pretty` | `v0.3.1` | MIT | 間接・開発用 |
| `github.com/kr/text` | `v0.2.0` | MIT | 間接・開発用 |
| `github.com/kylelemons/godebug` | `v1.1.0` | Apache-2.0 | 間接・開発用 |
| `github.com/mattn/go-isatty` | `v0.0.20` | MIT | 間接・開発用 |
| `github.com/miekg/dns` | `v1.1.66` | BSD-3-Clause | 間接・開発用 |
| `github.com/ncruces/go-strftime` | `v1.0.0` | MIT | 間接・開発用 |
| `github.com/remyoudompheng/bigfft` | `v0.0.0-20230129092748-24d4a6f8daec` | BSD-3-Clause | 実行時 |
| `github.com/rogpeppe/go-internal` | `v1.9.0` | BSD-3-Clause | 間接・開発用 |
| `golang.org/x/crypto` | `v0.53.0` | BSD-3-Clause | 間接・開発用 |
| `golang.org/x/exp` | `v0.0.0-20251023183803-a4bb9ffd2546` | BSD-3-Clause | 実行時 |
| `golang.org/x/mod` | `v0.29.0` | BSD-3-Clause | 間接・開発用 |
| `golang.org/x/net` | `v0.56.0` | BSD-3-Clause | 実行時 |
| `golang.org/x/sync` | `v0.17.0` | BSD-3-Clause | 間接・開発用 |
| `golang.org/x/sys` | `v0.46.0` | BSD-3-Clause | 実行時 |
| `golang.org/x/term` | `v0.44.0` | BSD-3-Clause | 間接・開発用 |
| `golang.org/x/text` | `v0.38.0` | BSD-3-Clause | 間接・開発用 |
| `golang.org/x/tools` | `v0.38.0` | BSD-3-Clause | 間接・開発用 |
| `golang.org/x/tools/go/expect` | `v0.1.1-deprecated` | BSD-3-Clause | 間接・開発用 |
| `golang.org/x/tools/go/packages/packagestest` | `v0.1.1-deprecated` | BSD-3-Clause | 間接・開発用 |
| `modernc.org/cc/v4` | `v4.27.1` | BSD-3-Clause | 間接・開発用 |
| `modernc.org/ccgo/v4` | `v4.30.1` | BSD-3-Clause | 間接・開発用 |
| `modernc.org/fileutil` | `v1.3.40` | BSD-3-Clause | 間接・開発用 |
| `modernc.org/gc/v2` | `v2.6.5` | BSD-3-Clause（同梱原文も参照） | 間接・開発用 |
| `modernc.org/gc/v3` | `v3.1.1` | BSD-3-Clause（同梱原文も参照） | 間接・開発用 |
| `modernc.org/goabi0` | `v0.2.0` | BSD-3-Clause | 間接・開発用 |
| `modernc.org/libc` | `v1.67.6` | BSD-3-Clause（同梱原文も参照） | 実行時 |
| `modernc.org/mathutil` | `v1.7.1` | BSD-3-Clause | 実行時 |
| `modernc.org/memory` | `v1.11.0` | BSD-3-Clause（同梱原文も参照） | 実行時 |
| `modernc.org/opt` | `v0.1.4` | BSD-3-Clause | 間接・開発用 |
| `modernc.org/sortutil` | `v1.2.1` | BSD-3-Clause | 間接・開発用 |
| `modernc.org/sqlite` | `v1.46.1` | BSD-3-Clause（同梱原文も参照） | 実行時 |
| `modernc.org/strutil` | `v1.2.1` | BSD-3-Clause | 間接・開発用 |
| `modernc.org/token` | `v1.1.0` | BSD-3-Clause | 間接・開発用 |
| `willnorris.com/go/microformats` | `v1.2.0` | MIT | 実行時 |
