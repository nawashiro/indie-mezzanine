# Spec Delta

## Purpose

購読者はcollection単位のAtomから検証済み投稿を読む。同じcollectionと投稿の識別子はリレーをまたいで一致し、読む側の将来の集約に使う。リレーは保存済みのmf2を正本としてAtomを生成する。

## ADDED Requirements

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
リレーはWebmentionで受け付けたsource URLをentry idとする SHALL。リレーは同じsourceの編集でentry idを変えない SHALL。リレーは投稿ごとにリレー固有のランダム識別子を生成しない SHALL。

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

### Requirement: 未知のcollectionと空のcollection
リレーは未知のcollectionと過去に投稿を持った空のcollectionを区別する SHALL。リレーは不正なUUID形式と未知のcollectionにHTTP 404を返す SHALL。リレーは既知の空collectionにentryなしのAtomを返す SHALL。

#### Scenario: 全投稿の撤回
- **WHEN** collectionの最後の投稿が撤回される
- **THEN** collectionのfeedはHTTP 200を返す
- **AND** feedは同じidを持ち、entryを含まない
