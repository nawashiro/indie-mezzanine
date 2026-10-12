# Design

## Context

動機は `proposal.md` を参照する。READMEとリモートリポジトリは新名を使う。Go実装、Compose、保存ファイルは旧名を使う。

初期changeのdesignはフィード著者名を `mezzanine relay` と定義する。本changeの `system-identity` はその表示を `indie-mezzanine` へ置き換える。

本changeはコード、配備、保存、文書を横断するためdesignを作成する。利用者は後方互換を不要とした。

## Goals / Non-Goals

**Goals:**
- 実装者は名前の対応表を使い、関連する参照を同時に更新する。
- 回帰テストは公開出力と配備入口の実値を検証する。
- 残存名の検査は製品名と外部識別子を区別する。

**Non-Goals:**
- 実装者は名称変更のために新しい設定項目や命名フレームワークを導入しない。
- 実装者はGoパッケージ名 `relay` を製品名へ変更しない。
- 実装者は旧資産の移行、稼働中サービスの再配備、ホストディレクトリの改名を実施しない。

## Decisions

### 1. 名前を固定し、対応表で統一する

実装者は次の対応を採用する。環境変数の製品接頭辞だけは大文字とアンダースコアで表す。

| 対象 | 統一後 |
| --- | --- |
| ルートのtitleとh1 | `indie-mezzanine` |
| Atomのフィード著者名 | `indie-mezzanine` |
| User-Agent | `indie-mezzanine Webmention receiver` |
| 起動ログ接頭辞 | `indie-mezzanine listening on` |
| Goモジュール | `indie-mezzanine` |
| 内部import | `indie-mezzanine/internal/relay` |
| コマンドソース | `cmd/indie-mezzanine/main.go` |
| ビルド出力 | `/out/indie-mezzanine` |
| コンテナ実行ファイル | `/indie-mezzanine` |
| Composeプロジェクト | `indie-mezzanine` |
| ローカルイメージ | `indie-mezzanine:local` |
| DB | `indie-mezzanine.db` |
| DNS隔離テストの環境変数 | `INDIE_MEZZANINE_DNS_CHILD` |
| Composeテストの保存先 | `/tmp/indie-mezzanine-compose-test-data` |

Goモジュールは既存の短いローカル名を改名する。実装者はリモートホスト名を新たにモジュールへ組み込まない。

表示名だけの変更は内部と運用の揺れを残すため採用しない。設定による製品名の差し替えも今回の固定名に不要なため採用しない。

### 2. DBを直接改名し、互換層を追加しない

`OpenStore` は新名のDBだけを開く。バックアップ復元テストは新名のDBを保存して読み戻す。

旧名ファイルだけが存在する場合も新名を使用する。テストは旧ファイルの内容を保持し、新名のストアが旧投稿を読み込まないことを確認する。

自動探索、移行、シンボリックリンク、フォールバックは採用しない。これらの処理は不要とした互換性を実装へ持ち込む。

### 3. 意味に基づいて文書と参照を更新する

実装者はREADMEの実行例、OpenSpec設定のProject、初期changeのproposalとdesignの現行製品名を更新する。設定のworking name表記を除去する。

実装者は次の参照を保持する。
- 概念名「中2階／中二階」と投稿例の題名。
- 記事URL `https://nawashiro.dev/posts/20261005-mezzanine`。
- 初期designの過去の一時資料パス。
- change識別子 `build-public-collection-relay` と既存のcapabilityパス。
- 役割名 `relay` と `cloudflared`、`PUBLIC_URL` など既存の設定変数。
- このchangeで旧名を説明する比較記述。

無条件の文字列置換は外部URLと証跡を壊すため採用しない。README以外の公開用文書は追加しない。

### 4. 挙動と残存参照を別々に検証する

Goテストはルート表示、Atomのフィード著者名、User-Agent、新名DBの作成と復元を確認する。既存テストは投稿者情報、UUID、配信経路の不変性を確認する。

Composeテストは解決後のプロジェクト名、イメージ名、ヘルスチェックを確認する。コンテナ試験は新名の実行ファイルとヘルスチェックを実行する。

残存名の検査は追跡ファイルのパスと内容を調べる。検査者は新名に含まれる `mezzanine` を旧名と誤判定しない。

検査者は旧名の完全語と `MEZZANINE_` 接頭辞を探す。検査者は保持対象、改名比較記述、旧名の非使用を確認するテストだけを例外として分類する。

## Risks / Trade-offs

- 新名DBは旧投稿を読み込まない → 後方互換なしの契約を明記し、旧ファイルを変更しないテストを追加する。
- Composeの新名は旧コンテナと併存する → 本changeは稼働中の配備を操作せず、旧資産を自動削除しない。
- Dockerのビルド先とヘルスチェックがずれる → Compose設定検査と実コンテナ試験を組み合わせる。
- 初期changeの設計と実装が食い違う → 現行製品名を更新し、両changeをOpenSpecで検証する。
- 残存検索が外部参照を改名する → 検査者は各残存参照を意味で分類する。

## Migration Plan

実装者は新名のビルドと空の保存先で検証する。旧DBからの移行手順と互換機構は提供しない。

本changeは既存配備の停止、削除、再起動を実行しない。ロールバック時の旧資産の利用は本changeの保証外とする。
