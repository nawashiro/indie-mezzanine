# ライセンス方針と依存の確認

## 本体はMITを採用する

mezzanineの独自コード、文書、実行設定はMIT Licenseで公開する。
著作権表示は `SHOYA Taichi (Nawashiro)` とする。
標準条文はルートの `LICENSE` に置く。
MITの再配布条件は著作権表示と許諾文の保持を要求する。[1]

第三者コードと第三者の権利表示には、それぞれの元の条件を適用する。
本体のMITは依存物をMITへ再ライセンスする宣言ではない。
配布者は本体の `LICENSE` と第三者通知を配布物に添える。
Dockerfileは `/LICENSE`、`/THIRD_PARTY_NOTICES.txt`、`/LICENSING.md` を同梱する。

この文書は依存と原文に基づく実務上の整理とする。
この文書は法的な保証を提供しない。

## Goバイナリに含まれる依存

確認対象はLinux amd64、非CGOビルドとする。
検証者は `go list` の実際の依存集合を確認した。
Go標準ライブラリのBSD条件は第三者通知に含む。
外部モジュールの集合は次の13件だった。

| モジュール | 固定バージョン | 主な条件 |
| --- | --- | --- |
| `github.com/doyensec/safeurl` | `v0.2.5` | Apache-2.0 |
| `github.com/dustin/go-humanize` | `v1.0.1` | MIT |
| `github.com/google/uuid` | `v1.6.0` | BSD-3-Clause |
| `github.com/gorilla/feeds` | `v1.2.0` | BSD-3-Clause |
| `github.com/remyoudompheng/bigfft` | `v0.0.0-20230129092748-24d4a6f8daec` | BSD-3-Clause |
| `golang.org/x/exp` | `v0.0.0-20251023183803-a4bb9ffd2546` | BSD-3-Clause |
| `golang.org/x/net` | `v0.56.0` | BSD-3-Clause |
| `golang.org/x/sys` | `v0.46.0` | BSD-3-Clause |
| `modernc.org/libc` | `v1.67.6` | BSD-3-Clause。同梱のMIT・BSD等の権利表示を保持 |
| `modernc.org/mathutil` | `v1.7.1` | BSD-3-Clause |
| `modernc.org/memory` | `v1.11.0` | BSD-3-Clause。同梱のGo・mmap-go権利表示を保持 |
| `modernc.org/sqlite` | `v1.46.1` | BSD-3-Clause。SQLite本体はpublic domain |
| `willnorris.com/go/microformats` | `v1.2.0` | MIT |

検証者は上記モジュールのライセンス原文と同梱の追加通知を確認した。
各原文は `THIRD_PARTY_NOTICES.txt` に全文が含まれた。
検証者は依存ソースへ変更を加えなかった。
BSD系の配布条件と著作権表示は元の原文を保持する。
Apache-2.0はライセンス本文の同梱などを要求する。[2]
safeurlの固定版には独立したNOTICEファイルがなかった。
SQLiteの配布コードは作者がpublic domainへ提供した。[6]
GoラッパーとSQLite本体の条件を混同しない。

## MPLの依存を切り分ける

全モジュールの一覧は `DEPENDENCIES.md` に置く。
一覧の `github.com/hashicorp/golang-lru/v2` はMPL-2.0を採用する。
`go mod why` は次の依存経路を示した。

```text
mezzanine/internal/relay
modernc.org/sqlite
modernc.org/libc
modernc.org/libc.test
modernc.org/ccgo/v4/lib
modernc.org/ccgo/v4/lib/internal/secret_sauce
modernc.org/gc/v3
github.com/hashicorp/golang-lru/v2
```

このモジュールは本番のGoバイナリに含まれなかった。
本リポジトリはそのソースをvendorとして再配布しない。
MPLはファイル単位の条件を持つ。独立した新規ファイルを一律にMPLへ変更しない。[3]
将来の依存変更でMPL対象が入る場合、配布者は対象ソースの提供条件を確認する。[5]

## コンテナの資産は別条件とする

Goバイナリとコンテナイメージの依存集合は異なる。
検証者は固定したDistrolessのlinux/amd64レイヤーを静的に読んだ。
検証者はコンテナを実行しなかった。
固定したイメージは次の値とする。

```text
gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
```

検証者はレイヤーのSHA-256を照合した。
イメージは以下のDebianパッケージ情報を保持した。

| パッケージ | 版 |
| --- | --- |
| `base-files` | `12.4+deb12u15` |
| `netbase` | `6.4` |
| `tzdata` | `2026b-0+deb12u1` |
| `media-types` | `10.0.0` |
| `ca-certificates` | `20250419~deb12u1` |

イメージは `/usr/share/doc/<package>/copyright` を保持した。
イメージは `/usr/share/common-licenses/` の条文も保持した。
`base-files` の権利表示はGPL-2-or-laterを示した。
`ca-certificates` はGPL-2-or-laterとMPL-2.0の対象を区別した。
Dockerfileはこれらを削除しない。
配布者はイメージ全体をMITだけと表示しない。

本リポジトリは本体ソースとDockerfileを公開する。
本リポジトリはベースイメージの中身をソースとして再配布しない。
配布者がビルド済みイメージを外部配布する場合、配布者は第三者資産の条件も確認する。
配布者は保持済みの権利表示と、対応するソースの入手導線を確認する。[5]
Debianの対応版ソースは次のパッケージアーカイブから確認する。

- https://snapshot.debian.org/package/base-files/
- https://snapshot.debian.org/package/netbase/
- https://snapshot.debian.org/package/tzdata/
- https://snapshot.debian.org/package/media-types/
- https://snapshot.debian.org/package/ca-certificates/

Goのビルド段は最終イメージへ丸ごとコピーしない。
最終イメージはビルド結果と配布用の文書だけを追加する。
コンテナ実行、他アーキテクチャ、将来の依存変更は今回の静的確認に含めない。

## OpenSpec生成スキル

`.hermes/skills/openspec-*` はOpenSpecの生成テンプレートを含む。
これらのテンプレートはMIT条件を持つ。
検証者はOpenSpec v1.14.1の元のMIT権利表示を第三者通知へ追加した。
OpenSpecテンプレートはGoバイナリにリンクしない。

## 再確認

```sh
CGO_ENABLED=0 go list -deps -json ./cmd/mezzanine
go list -m -json all
go mod why -m github.com/hashicorp/golang-lru/v2
go mod verify
```

配布者は実行対象、ライセンス原文、同梱通知の集合を照合する。
配布者は依存更新とベースイメージ変更でこの確認を繰り返す。

## Sources

[1] https://opensource.org/license/mit
    > "The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software."
[2] https://www.apache.org/licenses/LICENSE-2.0
    > "You must give any other recipients of the Work or Derivative Works a copy of this License;"
[3] https://www.mozilla.org/en-US/MPL/2.0/FAQ
    > "new files containing no MPL-licensed code are not Modifications"
[5] https://www.mozilla.org/en-US/MPL/2.0
    > "You may create and distribute a Larger Work under terms of Your choice, provided that You also comply with the requirements of this License for the Covered Software."
[6] https://www.sqlite.org/copyright.html
    > "All of the code and documentation in SQLite has been dedicated to the public domain by the authors."
