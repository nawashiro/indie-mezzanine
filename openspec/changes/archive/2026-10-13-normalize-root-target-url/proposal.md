# Proposal

## Why

リレーはHTTP(S)ルートURLの空パスと `/` を文字列比較で区別し、同じルートへの通知や原本リンクを拒否する。リレーは両表記の判定を統一し、自己通知の拒否も維持する。

## What Changes

- リレーは設定URL、通知target、解決済み原本リンクの空パスと `/` を同値とする。
- リレーは自己sourceの判定に同じ規則を使い、fragmentだけが異なる自己通知も拒否する。
- リレーは新規targetの内部表現に末尾 `/` を使う。
- リレーは比較時にも設定ルートと異なるtargetを引き続き拒否する。
- リレーはqueryと非空fragmentを除去してtargetを一致させない。
- リレーは任意sourceの入力表記と非ルートパスの末尾 `/` の意味を保持する。
- リレーは既存jobの両target表記を処理し、既存投稿とデータベースを一括書換えしない。
- 開発者は設定エラー文と運用説明に、ルートURLの両表記の受理を明示する。
- リレーは既存の設定受理範囲とURL解析・参照解決・直列化の動作を維持する。

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `webmention-ingestion`: ルートtargetの限定的な同値判定、自己通知の拒否、既存データとの互換性を明示する。

## Impact

- `internal/relay/server.go`: 通知受付、自己判定、新規jobのtarget表現に影響する。
- `internal/relay/parse.go`: 相対参照と基底URLの解決後のリンク比較に影響する。
- `internal/relay/config.go`: 開発者は検証条件を維持し、両表記を説明するエラー文へ修正する。
- `internal/relay/relay_test.go`、`internal/relay/retry_test.go`: 受付、解析、再起動の回帰検証を追加する。
- `.env.example`、`README.md`: 設定例の末尾 `/` を維持し、両表記の受理を説明する。

リレーはSQLiteスキーマ、sourceの一意制約、取得先の安全検査を変更しない。既存仕様はルート表記の同値性を明記せず、本変更はその判定を限定的に補う。

開発者は空の `#` 付き設定の新規拒否を別変更へ延期する。本変更は元のURL表記を追加検査し、既存の解析結果を硬化する変更を含まない。

検証失敗は当該jobを終了する。送信者は未保存sourceの問題を修正し、再通知する。過去の失敗は永久的な掲載拒否を意味しない。
