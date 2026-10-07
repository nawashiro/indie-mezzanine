# Spec Delta

## Purpose

公開リレーは投稿者のWebmentionを受け付け、リンク検証を通った投稿だけを掲載する。再通知は同じ投稿の更新と撤回を伝え、リレーは重複掲載と一時的な取得失敗による誤削除を防ぐ。

## ADDED Requirements

### Requirement: 公開Webmention受付
リレーは認証や手動承認なしでWebmentionを受け付ける SHALL。リレーはHTTP POSTのフォームからsourceとtargetを読み、受付対象のtargetと異なるsourceを要求する SHALL。リレーは受付成功と掲載成功を区別する SHALL。

#### Scenario: 正常な通知の受付
- **WHEN** 送信者が有効なsourceと設定済みのtargetを送る
- **THEN** リレーは処理を永続キューに登録してHTTP 202を返す
- **AND** リレーは検証前の投稿をfeedへ掲載しない

#### Scenario: 不正な通知
- **WHEN** sourceまたはtargetが欠落する、同値になる、またはtargetが受付対象ではない
- **THEN** リレーはHTTP 400を返す
- **AND** リレーは外部取得とキュー登録を行わない

### Requirement: 原本のリンク検証
リレーは安全な取得条件を通ったsourceのHTMLだけを検証する SHALL。リレーはWebmentionのリンク検証規則に従い、相対参照と文書の基底URLを扱う SHALL。リレーはcollection宣言だけをtargetへのリンクと見なさない SHALL。

#### Scenario: 所属と通知先の確認
- **WHEN** sourceがtargetへのリンクと有効なcollectionを持つ
- **THEN** リレーはh-entryとcollectionの条件を検査する
- **AND** 全条件を満たす場合だけリレーは投稿を掲載する

#### Scenario: 通知先へのリンクがない
- **WHEN** sourceがcollectionだけを持ち、targetへのリンクを持たない
- **THEN** リレーは投稿を掲載しない

### Requirement: 再通知の冪等性
リレーは同じsourceとtargetの再通知で投稿を重複させない SHALL。リレーは検証済みの新しい内容で同じ投稿を更新する SHALL。リレーは所属変更を旧collectionからの撤回と新collectionへの掲載として一貫して反映する SHALL。

#### Scenario: 同じ通知の再送
- **WHEN** 送信者が同じページを複数回通知する
- **THEN** feedは同じ投稿を一件だけ含む
- **AND** 内容が同じ場合、リレーは投稿の更新日時を進めない

#### Scenario: 投稿の編集と所属変更
- **WHEN** 再取得した投稿が別の有効なcollectionを一つ宣言する
- **THEN** リレーは旧collectionから投稿を除く
- **AND** リレーは新collectionへ更新済み投稿を掲載する

### Requirement: 撤回と一時失敗の区別
リレーはsourceのHTTP 404または410、targetリンクの消失、投稿資格の消失を撤回として扱う SHALL。リレーはタイムアウト、HTTP 5xx、安全な取得の拒否を一時失敗または処理失敗として記録し、直前の検証済み投稿を保持する SHALL。

#### Scenario: 投稿の削除通知
- **WHEN** 再通知したsourceがHTTP 404または410を返す
- **THEN** リレーは既存投稿をfeedから除く

#### Scenario: 公開ページから所属条件が消える
- **WHEN** 再取得に成功したsourceがtargetリンク、有効なcollection、または一意なh-entryを失う
- **THEN** リレーは既存投稿をfeedから除く

#### Scenario: 原本の一時障害
- **WHEN** 再通知の取得がタイムアウトする、HTTP 5xxを返す、または取得ポリシーに違反する
- **THEN** リレーは既存投稿を残す
- **AND** リレーは失敗状態を記録する

### Requirement: 受付資源の制限
リレーはリクエストサイズと未処理通知数に有限の上限を設ける SHALL。リレーは上限超過をHTTPエラーで通知し、受付済みと偽らない SHALL。

#### Scenario: キューの容量超過
- **WHEN** 永続キューが設定済み容量に達する
- **THEN** リレーは新規通知へHTTP 503を返す
- **AND** リレーは既存の検証済み投稿を保持する
