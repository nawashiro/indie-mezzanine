# Spec Delta

## ADDED Requirements

### Requirement: ブラウザアプリからのfeed購読
リレーはcollection別feedのHTTP応答に、任意オリジンのブラウザアプリが本文を読み取れるCORS許可を付す SHALL。許可は `Access-Control-Allow-Origin: *` とし、`Access-Control-Allow-Credentials` を出力しない MUST NOT。許可は本文を持つ200応答だけでなく、404・503のエラー応答にも及ぶ SHALL。

#### Scenario: 投稿を含むfeedのブラウザ取得
- **WHEN** 任意のオリジンのブラウザアプリが既存collectionの `.atom` または `.jf2` を取得する
- **THEN** 応答はHTTP 200と `Access-Control-Allow-Origin: *` を持つ
- **AND** 応答は `Access-Control-Allow-Credentials` を持たない

#### Scenario: 未知のcollection
- **WHEN** ブラウザアプリが不正なUUID形式または未知のcollectionのfeedを取得する
- **THEN** 応答はHTTP 404と `Access-Control-Allow-Origin: *` を持つ
- **AND** ブラウザアプリは404というステータスを読み取れる

#### Scenario: 保存データの取得失敗
- **WHEN** ブラウザアプリがfeedを取得し、保存データの取得に失敗する
- **THEN** 応答はHTTP 503と `Access-Control-Allow-Origin: *` を持つ
