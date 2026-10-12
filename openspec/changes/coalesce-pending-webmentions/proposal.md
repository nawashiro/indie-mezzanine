# Proposal

## Why

現行のリレーは、同じsourceの処理待ち通知ごとにjobを追加する。issue #3は、重複通知によるキュー占有と不要な後続処理を問題として扱う。

## What Changes

- リレーは、同じsourceの`pending`または`running`のjobへ再通知を集約する。
- リレーは、永続化済みの処理待ちjobを確認してHTTP 202を返す。リレーは追加のjobやキュー枠を作らない。
- リレーは、容量判定より先に処理待ち重複を確認する。満杯でも既存jobへの再通知は成功する。
- リレーは、再通知で既存jobのtarget、試行回数、再試行予定、受付時刻を変更しない。
- リレーは、終端失敗後の新しい通知を通常の容量判定へ戻す。
- リレーは、保存済みsourceのHTTP 200とスナップショット不変性を維持する。
- 本変更は、公開URLの正規化を扱わない。別変更`normalize-root-target-url`は、その仕様を扱う。

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `webmention-ingestion`: 処理待ち通知のsource単位の冪等受付を追加する。公開受付と容量超過の条件を明確にする。

## Impact

- `internal/relay/store.go`: `Enqueue`は、保存済み判定と容量判定の間で処理待ちjobを確認する。
- `internal/relay/server.go`: Webmention受付は、処理待ちの再通知へ掲載成功と異なる応答を返す。
- `internal/relay/relay_test.go`: `TestQueue`と`TestHTTPAdmission`は、重複を容量消費として数える現行期待値を変更する。
- `internal/relay/retry_test.go`と`contracts_test.go`: 回帰検証は、有限再試行、復旧、保存結果の不変性を確認する。
- 公開受付の応答は、処理待ち重複の場合に503から202へ変わる。満杯時の異なる未保存sourceは503を維持する。
- 本変更は、依存追加、スキーマ変更、既存の重複jobの移行を行わない。旧jobは従来のライフサイクルで処理と期限削除を続ける。
