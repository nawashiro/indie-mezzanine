# collection-feeds Specification

## Purpose

購読者はcollection単位のAtomから、ビューワーはcollection単位のJF2 Feedから、同じ検証済み投稿を読む。同じcollectionと投稿の識別子はリレーをまたいで一致し、読む側の将来の集約に使う。リレーは保存済みのmf2を正本として両形式を生成する。

## Requirements

### Requirement: collection別のAtom
リレーはcollection別の公開AtomをHTTP GETで提供する SHALL。feedはAtom名前空間を使い、idをcollectionの正規UUID URN、selfリンクをそのリレーの配信URLとする SHALL。リレーはアカウントや購読者認証を要求しない SHALL。

#### Scenario: 投稿を含むfeed
- **WHEN** 購読者が既存collectionのfeedを取得する
- **THEN** リレーはHTTP 200と `application/atom+xml` を返す
- **AND** feedはそのcollectionの検証済み投稿だけを含む

#### Scenario: 別のリレーの同じcollection
- **WHEN** 二つのリレーが同じcollectionを配信する
- **THEN** 両feedは同じidを持つ
- **AND** 各feedは各リレーのURLをselfリンクに使う

### Requirement: 安定した投稿識別子
リレーはWebmentionで受け付けたsource URLをentry idとする SHALL。リレーは同じsourceの原本編集と再通知で保存済みentryを変えない SHALL。リレーは投稿ごとにリレー固有のランダム識別子を生成しない SHALL。

#### Scenario: 複数リレーへの同じ投稿
- **WHEN** 投稿者が同じsourceを二つのリレーへ通知する
- **THEN** 両リレーのentryは同じidを持つ

#### Scenario: 原本のリダイレクト
- **WHEN** sourceの安全な取得が別のURLへリダイレクトする
- **THEN** entry idは受付時のsource URLを保持する
- **AND** リレーは最終取得URLを解析の基底URLとして扱う

### Requirement: テキストだけのAtomコンテンツ
リレーは保存済みmf2のプレーンテキストをAtom contentのtype=textで配信する SHALL。リレーはXMLの文字データとして本文をエスケープする SHALL。リレーはHTML表現を復元しない SHALL。

#### Scenario: タグに見える文字列
- **WHEN** コンテンツのvalueが `<script>alert(1)</script>` とアンパサンドを含む
- **THEN** Atomの本文はtype=textになる
- **AND** XML解析後の本文は同じ文字列になる
- **AND** 本文はscriptの子要素を含まない

### Requirement: 配信メタデータと順序
リレーは各feedとentryにAtomが要求するメタデータを出力する SHALL。リレーはmf2のタイトル、著者、日時を利用し、欠落時の明示的な代替値を使う SHALL。リレーはentry.updatedの降順とentry idの昇順で順序を固定し、配信件数に上限を設ける SHALL。

#### Scenario: メタデータが少ない投稿
- **WHEN** 有効なh-entryがタイトル、著者、公開日時を持たない
- **THEN** リレーは定義済みの代替値で有効なAtom entryを生成する
- **AND** リレーは内容と元ページへのリンクを配信する

#### Scenario: 再取得の順序
- **WHEN** 内容が変わらないfeedを複数回取得する
- **THEN** entryの順序とupdatedは変わらない

### Requirement: 保存済みスナップショットからの配信
リレーは原本を再取得せず、保存済みスナップショットからfeedを生成する SHALL。リレーは再通知でentryの本文、所属、メタデータ、updatedとfeed.updatedを変更しない SHALL。リレーは不正なUUID形式と未知のcollectionにHTTP 404を返す SHALL。

#### Scenario: 原本の編集と削除
- **WHEN** 保存後に原本が編集または削除され、同じsourceが再通知される
- **THEN** 元のcollectionは保存済みの同じentryを配信する
- **AND** entryとfeedのupdatedは再通知で変わらない

#### Scenario: 未知のcollection
- **WHEN** 初回保存に成功した投稿がないcollectionのfeedを取得する
- **THEN** リレーはHTTP 404を返す

### Requirement: collection別のJF2 Feed
リレーはcollection別のJF2 FeedをHTTP GETで提供する SHALL。feedはJF2 Feedプロファイルに準拠し、トップレベルを `type: "feed"`、投稿を `children` の `type: "entry"` とし、各entryに単一文字列の `uid` を持たせる SHALL。リレーは `application/jf2feed+json` をcharsetなしで返す SHALL。feedの `name` をcollectionの正規URN、`url` をそのリレーのJF2 FeedのURL、`author` を `indie-mezzanine` のcardとする SHALL。リレーはアカウントや購読者認証を要求しない SHALL。リレーは不正なUUID形式と未知のcollectionにHTTP 404を返す SHALL。

#### Scenario: 投稿を含むJF2 Feed
- **WHEN** ビューワーが既存collectionのJF2 Feedを取得する
- **THEN** リレーはHTTP 200と `application/jf2feed+json` を返す
- **AND** feedは `type: "feed"` とそのcollectionの検証済み投稿だけを `children` に含む
- **AND** 各entryは `type: "entry"` と単一文字列の `uid` を持つ

#### Scenario: 未知のcollection
- **WHEN** ビューワーが不正なUUID形式または未知のcollectionのJF2 Feedを取得する
- **THEN** リレーはHTTP 404を返す

#### Scenario: 別のリレーの同じcollection
- **WHEN** 二つのリレーが同じcollectionのJF2 Feedを配信する
- **THEN** 両feedの `name` は同じ正規URNとなる
- **AND** 各feedの `url` は各リレーのJF2 FeedのURLとなる
- **AND** 両feedの同じ投稿の `uid` は一致する

### Requirement: JF2の安定した投稿識別子
リレーはWebmentionで受け付けたsource URLをentryの `uid` と `url` とする SHALL。リレーは保存済みmf2の `uid` と `url` をentryの識別子とリンクに使用しない MUST NOT。リレーは同じsourceの原本編集と再通知で保存済みentryを変えない SHALL。リレーは投稿ごとにリレー固有のランダム識別子を生成しない SHALL。

#### Scenario: 複数リレーへの同じ投稿
- **WHEN** 投稿者が同じsourceを二つのリレーへ通知する
- **THEN** 両リレーのentryは同じ `uid` と `url` を持つ

#### Scenario: 原本のリダイレクト
- **WHEN** sourceの安全な取得が別のURLへリダイレクトする
- **THEN** `uid` は受付時のsource URLを保持する
- **AND** リレーは最終取得URLを解析の基底URLとして扱う

#### Scenario: 原本の編集と再通知
- **WHEN** 保存後に原本が編集または削除され、同じsourceが再通知される
- **THEN** entryの `uid` と本文は変わらない

### Requirement: テキストだけのJF2コンテンツ
リレーは保存済みmf2のプレーンテキストをJF2の `content.text` としてだけ配信する SHALL。リレーはHTML表現を出力しない MUST NOT。リレーは本文を持たない投稿に空の `content` を生成しない SHALL。

#### Scenario: タグに見える文字列
- **WHEN** コンテンツのvalueが `<script>alert(1)</script>` とアンパサンドを含む
- **THEN** 本文は `content.text` にだけ現れる
- **AND** JSON解析後の本文は同じ文字列になる
- **AND** `content` は `html` キーを持たない

#### Scenario: 本文の欠落
- **WHEN** 保存済みmf2がcontentを持たない
- **THEN** リレーは空の `content` を生成しない
- **AND** entryは他のメタデータだけで有効なJF2 entryとなる

### Requirement: JF2のメタデータと順序
リレーはentryの `name` に保存済みmf2の名称を使い、欠落時はsource URLとする SHALL。リレーはentryの `author` をcardとし、名称の欠落時に定義済みの代替値を使う SHALL。リレーは有効な日時だけを `published` に出力し、`updated` は保存済みの日時、欠落時は初回保存日時とする SHALL。リレーはentry.updatedの降順とentry idの昇順で順序を固定し、配信件数に上限を設ける SHALL。

#### Scenario: メタデータが少ない投稿
- **WHEN** 有効なh-entryが名称、著者、公開日時を持たない
- **THEN** `name` はsource URLとなり、`author` は定義済みの代替値を持つ
- **AND** `published` を出力せず、`updated` は初回保存日時となる

#### Scenario: 再取得の順序
- **WHEN** 内容が変わらないJF2 Feedを複数回取得する
- **THEN** entryの順序と日時文字列は変わらない

### Requirement: JF2の正規化と連絡先の非公開
リレーはtypeから `h-` 接頭辞を除去し、単一要素の配列を単一値へ畳む SHALL。リレーは `category` を文字列の配列、`photo` を単一の文字列（複数ある場合は先頭）、`video` と `audio` を `url` を持つオブジェクトとして出力する SHALL。リレーは入れ子オブジェクトの `value` を出力しない SHALL。リレーは保存済みmf2の `email` を出力しない MUST NOT。

#### Scenario: 複数の写真
- **WHEN** entryが複数のphotoを持つ
- **THEN** `photo` は先頭URLの単一文字列となる
- **AND** 残りのphotoは出力されない

#### Scenario: 入れ子の著者とchildren
- **WHEN** h-entryが入れ子のh-card著者とchildrenを持つ
- **THEN** 各typeは `h-` を除いた単一文字列となる
- **AND** 入れ子オブジェクトの `value` は出力されない
- **AND** `category` は文字列の配列となる

#### Scenario: 連絡先プロパティ
- **WHEN** 原本のh-cardにu-emailがある
- **THEN** entryとその `author` に `email` は現れない

### Requirement: 形式間の一貫性
リレーは同じ保存済みスナップショットからAtomとJF2 Feedを生成し、両形式で同じentry集合と同じ順序を配信する SHALL。同じ投稿のAtom entry idとJF2の `uid`、および対応する日時文字列は一致する SHALL。リレーはJF2の追加によって既存Atomの応答を変更しない SHALL。リレーは再通知で両形式のentry本文、所属、メタデータ、日時を変更しない SHALL。

#### Scenario: 同じcollectionの両形式
- **WHEN** 購読者とビューワーが同じcollectionのAtomとJF2 Feedを取得する
- **THEN** 両形式は同じentry集合と同じ順序を持つ
- **AND** 同じ投稿のentry idと `uid`、対応する日時文字列は一致する

#### Scenario: 既存Atomの維持
- **WHEN** リレーがJF2 Feedを追加して配信する
- **THEN** Atomエンドポイントの応答はJF2追加前と変わらない

#### Scenario: 再通知の不変
- **WHEN** 保存済みsourceが再通知される
- **THEN** 両形式のentry本文、所属、メタデータ、日時は変わらない

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
