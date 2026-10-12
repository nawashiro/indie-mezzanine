# Proposal

## Why

ブラウザアプリ（自作ビューワー、汎用Webリーダー）は、CORS応答ヘッダがないためブラウザに応答を拒否され、collection別のAtomとJF2 Feedを読めない。feedは認証なしで誰でも読める公開データであり、オリジン依存の秘密が存在しない。CORSでブラウザアプリをはじくことに保護上の意味はなく、消費者を減らすだけだ。jf2追加時にビューワー配置が未定だったため先送りしたCORS判断に、読み取り専用のブラウザ消費者という答えを出す。

## What Changes

- `GET /collections/{name}` のfeed応答に `Access-Control-Allow-Origin: *` を付す。成功応答（`.atom` / `.jf2` の200）だけでなく、404（不正UUID・未知collection・未対応接尾辞）と503（保存データ取得失敗）にも付す。
- `Access-Control-Allow-Credentials` は付けない。任意オリジンからの認証情報付きリクエストを許可しない。
- 許可オリジンの設定項目は追加しない（`*` 固定）。`Vary` は不要。
- 対象外: プリフライト（OPTIONS）、ETag/Last-Modifiedと304、`POST /webmention` 応答のCORS、`rel="alternate"` の発見経路、ランディングページ。

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `collection-feeds`: collection別feedの応答に、任意オリジンのブラウザアプリが読み取れるCORS許可を付す要件を追加する。許可はエラー応答（404・503）にも及ぶ。認証情報付きの許可は出力しない。

## Impact

- コード: `internal/relay/server.go` の `GET /collections/{name}` ハンドラに応答ヘッダの設定を追加する。
- テスト: `internal/relay/contracts_test.go` と `internal/relay/jf2_test.go` のHTTP契約テストにヘッダ検証を追加する。
- API契約: feed応答のヘッダが増える。ステータス、本文、保存データ、スキーマ、設定は不変。
- 運用: 既存配備のまま有効。ロールバックは以前のイメージへ戻すとヘッダが消えるだけで、応答本文と保存データは不変。
