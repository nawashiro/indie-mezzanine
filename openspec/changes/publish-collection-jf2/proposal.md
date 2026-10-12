# Proposal

## Why

別途用意するビューワーは、著者情報や写真などの構造を必要とする。現行Atomは本文textのみで、著者写真はAtomのauthorがname/uri/emailしか持たないため表現できない。保存済みスナップショット(HTML除去済みmf2)にはその構造が残っているので、そのJSON射影を第2の出口として公開すれば、リレーにUIを持ち込まずにビューワーを成立させられる。

## What Changes

- `GET /collections/{uuid}.jf2` を追加する。JF2 Feedプロファイル(W3C Note jf2のJF2 Feed)に準拠し、`application/jf2feed+json` で返す。
- 保存済みmf2の機械的正規化を射影規則として固定する。`uid`/`url`はsource URL、`content`は`{"text": ...}`のみ、単一要素配列の畳み込み、`h-`接頭辞の除去、`category`は文字列配列、`photo`は単一文字列(先頭のみ)、`email`は出力しない。
- Atomと同一の不変条件をjf2にも適用する。原本の再取得なし、再通知で不変、順序(entry.updated降順・entry id昇順)と上限、不正UUID・未知collectionは404、認証なし、決定的出力。
- 形式間の一致を要求する。同一スナップショットから両形式を生成し、entry集合・順序・識別子・日時文字列が一致する。JF2追加によって既存Atomの出力を変更しない。
- 対象外: ビューワー本体、CORS許可、発見経路(`rel="alternate"`)、HTMLマークアップ生成、画像の取得・プロキシ、段落境界の再構成、スナップショットとスキーマの変更。

## Capabilities

### New Capabilities

- なし

### Modified Capabilities

- `collection-feeds`: collection単位の配信にJF2 Feedを追加し、保存済みスナップショットからの射影規則、連絡先の非公開、形式間の一致を要求する。Purposeを両形式に更新する。

## Impact

- コード: `internal/relay/feed.go` と並ぶJF2生成の追加と、`internal/relay/server.go` の `GET /collections/{name}` への `.jf2` 分岐。保存モデル、スキーマ、設定(`FEED_LIMIT`)は不変。
- 依存: 新規依存なし(`encoding/json`)。第三者通知の更新は不要。
- 文書: READMEへjf2の購読URLとメディアタイプを追記。ビューワーとCORSが対象外であることも記す。
- テスト: 契約テストへプロファイル準拠、決定性、形式間一致、404、非公開の検査を追加。
- 運用: 既存配備のまま有効。ロールバックはイメージ差し替えで、jf2経路が消えるだけ。Atom応答と保存データは不変。
