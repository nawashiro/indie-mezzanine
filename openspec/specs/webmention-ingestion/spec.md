# webmention-ingestion Specification

## Purpose

公開リレーはWebmentionを受け付け、取得とリンク検証に成功した原本をスナップショットとして掲載する。リレーはsource URLごとの最初の検証成功結果を固定し、再通知による上書き、所属移動、撤回を行わない。

## Requirements

### Requirement: 公開Webmention受付
リレーは認証や手動承認なしでWebmentionを受け付ける SHALL。リレーはHTTP POSTのフォームからsourceとtargetを読み、受付対象のtargetと異なるsourceを要求する SHALL。リレーは受付成功と掲載成功を区別する SHALL。

#### Scenario: 未保存sourceの正常な通知
- **WHEN** 送信者が未保存の有効なsourceと設定済みのtargetを送る
- **THEN** リレーは処理を永続キューに登録してHTTP 202を返す
- **AND** リレーは検証前の投稿をfeedへ掲載しない

#### Scenario: 不正な通知
- **WHEN** sourceまたはtargetが欠落する、同値になる、またはtargetが受付対象ではない
- **THEN** リレーはHTTP 400を返す
- **AND** リレーは外部取得とキュー登録を行わない

### Requirement: 保存前の原本検証
リレーは未保存sourceの安全に取得したHTMLからtargetリンクと投稿資格を検査する SHALL。リレーは相対参照と文書の基底URLを扱う SHALL。リレーはcollection宣言だけをtargetリンクと見なさない SHALL。スナップショットは通知後の取得・検証結果とし、通知到着時刻の原本を復元する機能を提供しない SHALL。

#### Scenario: 初回の掲載
- **WHEN** 取得したsourceがtargetリンク、一意なh-entry、有効な単一collectionを持つ
- **THEN** リレーは検証結果を初回スナップショットとして保存する
- **AND** 保存後にだけfeedへ掲載する

#### Scenario: 未保存sourceの検証失敗
- **WHEN** sourceが404または410を返す、またはtargetリンクや投稿資格を持たない
- **THEN** リレーはスナップショットを作らない
- **AND** リレーはその通知の検証失敗を記録する

#### Scenario: 保存前の一時障害
- **WHEN** 未保存sourceの取得がタイムアウト、5xx、安全拒否、本文上限超過で失敗する
- **THEN** リレーはスナップショットを作らない
- **AND** リレーは失敗を記録し、再試行を有限にする

#### Scenario: 初回失敗後の新しい通知
- **WHEN** 未保存sourceの通知が失敗し、後の通知で初めて取得と検証に成功する
- **THEN** リレーはその成功結果を最初のスナップショットとして保存する
- **AND** 過去の失敗だけを理由に永久拒否しない

### Requirement: 保存済みsourceの再通知を無視
リレーはsource URLごとに最初の検証成功スナップショットを一件だけ保持する SHALL。有効なフォームが保存済みsourceを通知した場合、リレーはHTTP 200を返し、外部再取得、新しいjobの登録、投稿やcollectionの変更を行わない SHALL。原本の編集、所属変更、削除、資格消失は保存結果へ反映しない SHALL。

#### Scenario: 保存後の再通知
- **WHEN** 保存済みsourceが同じ内容、編集済み内容、別のcollectionで再通知される
- **THEN** リレーは最初の本文、所属、識別子、保存日時を保持する
- **AND** リレーはsourceへHTTPリクエストを送らない
- **AND** リレーは通知履歴やスナップショットを増やさない

#### Scenario: 原本削除後の再通知
- **WHEN** 保存済みsourceが削除または資格消失後に再通知される
- **THEN** リレーは原本を再取得せず、最初のスナップショットを保持する
- **AND** リレーは撤回や所属移動を行わない

#### Scenario: 同時通知と保存後の中断復旧
- **WHEN** 同じsourceの通知が同時に届く、またはスナップショット保存後のjobが再処理される
- **THEN** リレーは保存結果を上書きせず、一件のスナップショットを保持する
- **AND** 保存済みsourceを処理するworkerは外部取得を行わない

### Requirement: 受付資源の制限
リレーはリクエストサイズと未処理通知数に有限の上限を設ける SHALL。リレーは新規通知の上限超過をHTTPエラーで通知し、受付済みと偽らない SHALL。

#### Scenario: キューの容量超過
- **WHEN** 永続キューが容量に達し、未保存sourceの通知が届く
- **THEN** リレーはHTTP 503を返す
- **AND** リレーは保存していない通知を受付済みと偽らない

#### Scenario: 満杯時の保存済みsource
- **WHEN** 永続キューが容量に達し、有効なフォームが保存済みsourceを通知する
- **THEN** リレーはHTTP 200を返し、追加のキュー容量を使わない
