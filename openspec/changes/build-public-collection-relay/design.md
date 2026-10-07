# Design

## Context

動機と範囲はproposal.mdを参照する。リポジトリはアプリケーションコードを持たない。

部品検証はmf2の所属先、入れ子、HTML除去、JSON往復、Atomのtext出力を確認した。最終実行は31テストを5回通した。検証はSQLite、Docker、Webmention受付全体を含まない。

再実行用の一時資料は `/opt/data/cache/mezzanine-spike-BNrG9V/README.md` にある。このパスは永続の依存先にしない。実装者は独立したfixtureと必要な検証をリポジトリへ作成する。

合意済みの境界は公開リレー、単一コンテナ、ページ内のh-entry一つ、collection一つ、UUID URN限定、内部mf2、HTML表現の除去とする。以下のHTTP経路、代替メタデータ、資源上限はレビュー対象の具体案とする。

## Goals / Non-Goals

**Goals:**
- アプリは取得、解析、保存、配信を分離する。
- アプリは解析前のHTMLだけを一時メモリで扱う。
- アプリは受付済み通知を永続化し、再起動後に再処理する。
- アプリは少数の既存部品を固定し、独自コードを接着層へ限定する。

**Non-Goals:**
- アプリは受信済みページを自動巡回しない。投稿者は更新・削除を再通知する。
- アプリは外部著者ページ、短縮URL、画像を取得しない。
- アプリはmf2の段落を独自HTMLレンダラーで復元しない。
- アプリはクラスタ運用、横断検索、複数collectionへの所属を提供しない。

## Decisions

### 1. 単一GoプロセスとSQLite

HTTPサーバーと有限数のworkerを同じプロセスで動かす。SQLiteは投稿と通知キューを保存する。別DBとRedisは運用対象を増やすため採用しない。

SQLiteドライバーはpure Goの `modernc.org/sqlite` を第一候補とする。実装者は依存導入、非CGOビルド、永続化、コンテナ起動を検証する。採用失敗時は実装者が設計変更を提示し、別ドライバーへ無言で切り替えない。

ストアは次の情報を持つ。
- collections: UUID値、正規URN、feed更新日時。
- posts: source URL、target URL、最終取得URL、collection、HTML除去済みmf2 JSON、内容ハッシュ、初回掲載日時、更新日時。
- jobs: source、target、状態、再試行情報、有限の処理期限。

SQLiteの一意制約はsourceとtargetの重複掲載を防ぐ。workerは同じsourceの処理を直列化する。掲載と所属変更は一つのトランザクションで反映する。

### 2. 標準HTTPサーバーと公開URL

HTTPサーバーはGo標準の `net/http` を使う。初版は次の経路を持つ。

| 経路 | 役割 |
| --- | --- |
| `GET /` | リレー説明とWebmention endpointの広告 |
| `POST /webmention` | sourceとtargetのフォーム受付 |
| `GET /collections/{uuid}.atom` | collection別Atom |
| `GET /healthz` | HTTP、保存先、workerの稼働確認 |

運用者は `PUBLIC_URL` をリレーの公開ルートURLとして指定する。初版のtargetはこのURL一つとする。アプリはHostヘッダーから公開URLを推測しない。ルートはHTTP LinkヘッダーとHTMLのrel=webmentionでendpointを広告する。

`PUBLIC_URL` はHTTPまたはHTTPS、認証情報なし、クエリなし、フラグメントなし、ルートパスだけを許可する。サブパス配備は初版の対象外とする。コンテナはHTTPで待ち受け、公開HTTPSの終端は既存の配備環境に任せる。

### 3. 取得を一か所に限定する

取得層は `doyensec/safeurl` を使う。初期候補バージョンは部品検証と同じv0.2.5とする。実装者は依存の安全性と互換性を再確認する。

取得層はHTTPとHTTPS、80と443、URL内認証情報なしを要求する。取得層はライブラリの接続時IP検査を維持する。取得層は環境プロキシを無効化する。

部品検証は認証情報付きLocationがデフォルトのredirect経路を通る問題を再現した。専用CheckRedirectは各転送先のスキーム、ポート、認証情報、回数を検査する。接続先IPの拒否は取得ライブラリに任せる。

部品検証はsafeurlのテストモード専用DNSカウンターにデータ競合を検出した。本番はテストモードを禁止する。テストは取得層の依存注入か隔離した検証プロセスを使い、グローバルresolverを書き換えない。

初期上限案は次の値とする。これらは検証fixtureの値ではない。
- 取得全体: 10秒。
- 展開後本文: 1 MiB。
- リダイレクト: 最大5回。
- 同時取得: 2件。
- 受付フォーム: 16 KiB。
- 未処理通知: 1000件。
- 一つのfeed: 最大100件。

取得層は上限を設定として検査する。0以下などの無制限指定は起動エラーとする。

### 4. 取得した同じHTMLでリンクとmf2を解析する

解析層は既存HTMLパーサーと `willnorris.com/go/microformats` を使う。初期mf2候補バージョンはv1.2.0とする。解析層は最終取得URLと文書のbaseを参照解決に使う。

リンク検証の小さな接着層はWebmention仕様の対象属性と参照解決を扱う。実装者はWebmentionの検証fixtureと相互運用テストでこの責務を確認する。検証済みでない独自仕様をWebmention準拠と呼ばない。

mf2解析層は返却JSONのitemsとchildrenを走査し、h-entryが一つであることを検査する。h-cardなどの著者データはそのh-entry内の構造を保つ。ページ全体の無関係な項目は投稿に含めない。

mf2のrels.collectionはページ単位の所属先とする。同一UUIDの重複宣言は一つと数える。異なる所属先が複数、または不正な宣言が混在する場合、解析層は投稿資格なしと判定する。

### 5. UUID URNを先に検査し、後でUUID値へ変換する

入力検査は小文字の `urn:uuid:` 接頭辞と、8-4-4-4-12桁の16進UUIDだけを許可する。検査層は空白を除去せず、短い形式を補完しない。

UUIDパーサー単独の受理条件に依存しない。入力の形を検査した後、既存UUIDライブラリで値へ変換する。初版は特定のUUIDバージョンだけへ限定しない。

ストアはUUID値で同一性を扱う。公開feed idとURLのuuid部分は小文字へ正規化する。原入力の受理と、検査済み値の表示形式を混同しない。

### 6. HTMLを除いたmf2を内部の正本にする

保存境界は解析済みmf2から全階層のHTML表現を除去する。保存境界はe-contentのvalue、type、properties、children、入れ子の著者を保持する。DOMと取得HTMLは保存対象外とする。

内容ハッシュは安定したJSON表現と所属情報から作る。同じ内容の再通知はupdatedを進めない。取得日時と投稿の更新日時は別の値として扱う。

部品検証のmf2 valueは一部の段落境界を詰めた。初版は既存パーサーのvalueを採用する。改行の再構成は別変更とする。

### 7. 更新と撤回を再通知から反映する

受信層はフォーム検査後にjobを保存し、HTTP 202を返す。workerは取得と解析を行い、成功時に投稿をupsertする。

workerはHTTP 404または410、取得成功後のtargetリンク消失、h-entryやcollectionの資格消失で掲載を撤回する。既知のcollection記録は残し、空feedを配信する。

タイムアウト、HTTP 5xx、サイズ超過、安全な取得の拒否は既存投稿を消さない。workerは失敗状態を記録する。再試行回数と保持期間は有限にする。初期案は最大3回、処理済みjobの保持7日とする。

再起動は未完了jobを再処理対象へ戻す。無制限の再試行と、処理済みjobの無期限蓄積は採用しない。

### 8. Atomは保存モデルから作る出口とする

出力層は `gorilla/feeds` のAtomFeed、AtomEntry、AtomContentを直接組み立てる。初期候補バージョンはv1.2.0とする。出力層は一般Feed変換経路を使わず、本文にtype=textを明示する。

entry idは受付時のsource URLとする。u-urlやリダイレクト先URLはentry idを書き換えない。entryの元ページリンクはsource URLとする。

メタデータの代替案は次の規則とする。
- feed title: 正規collection URN。
- feed author: 「mezzanine relay」。各entryは投稿者のauthorを別に持つ。
- entry title: mf2のname。欠落時はsource URL。
- author: 入れ子の著者name。欠落時は表示名「投稿者不明」。
- published: 有効なmf2日時がある場合だけ出力する。
- updated: 有効なmf2 updated。欠落時は内容変更の反映日時。
- 空本文: 空のtextとして扱う。ページ全体から補完しない。

同じ投稿への再通知はentry idを保つ。feed.updatedは所属投稿の掲載、内容変更、撤回時にだけ進める。出力層は取得のたびに現在時刻を生成しない。

### 9. 小さい構成と独立fixture

実装の境界はHTTP、取得、mf2検査、ストア、Atomとする。実装者は必要以上のプラグイン基盤と汎用フレームワークを導入しない。

回帰テストは固定fixtureを使う。回帰テストは記事の文言、記事ID、実記事の要素数を固定しない。将来のGUIは保存済みmf2を使い、テキスト表示とURLの安全性を別に検査する。

## Risks / Trade-offs

- 公開受付のスパムとDoS → 有限の受付・キュー・取得上限を置く。認証や承認UIは追加しない。
- 外部URL取得のSSRF（Server-Side Request Forgery） → 接続時IP検査と各redirectのURL検査を併用する。
- 部品の安全性を過信する危険 → 本番設定で圧縮本文、proxy、DNS、redirectの回帰テストを実行する。
- 独自Webmention接着層の相互運用不足 → 仕様由来のfixtureと公開相互運用ケースで確認する。
- mf2のHTML除去が万能なXSS対策に見える危険 → 本文はtextで出す。GUIとURL表示は別変更で検査する。
- source URLをidとするためURL変更を別投稿と数える → 初版は同一sourceの再通知だけを更新とする。
- 削除は再通知まで反映しない → 運用説明はこの制約を明記する。
- SQLiteとコンテナの未検証 → 実装タスクは再起動・保存・非root運用の実行確認を含む。

## Migration Plan

新規アプリのため既存データ移行はない。初回起動はスキーマを初期化する。更新はスキーマ版を検査する。

運用者は停止後のSQLiteファイルをバックアップする。運用者は以前のイメージと互換な保存データでロールバックする。不可逆なスキーマ更新は別変更で扱う。

## Open Questions

仕様と実装タスクを変える未解決事項は残さない。本書の具体案は実装開始前のレビュー対象とする。TLS終端の製品と公開ドメインの実値は配備時に選ぶ。
