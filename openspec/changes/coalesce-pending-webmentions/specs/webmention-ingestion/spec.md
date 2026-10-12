# Spec Delta

## ADDED Requirements

### Requirement: 処理待ちsourceの再通知を集約
リレーは、source URLの同じ文字列を処理待ち通知の同一性として使う SHALL。リレーは、有効な再通知を同じsourceの`pending`または`running`のjobへ集約する SHALL。リレーは、再通知で新しいjob、追加のキュー占有、追加の取得処理を作らない SHALL。リレーは、再通知で既存jobのtarget、状態、試行回数、再試行予定、受付時刻を変更しない SHALL。リレーは、有限再試行と中断復旧を維持する SHALL。

#### Scenario: 処理開始前の再通知
- **WHEN** 送信者が`pending`のjobを持つ未保存sourceを有効なフォームで再通知する
- **THEN** リレーはHTTP 202を返す
- **AND** リレーは既存jobを保持し、新しいjobを追加しない
- **AND** リレーは既存jobのtarget、状態、試行回数、再試行予定、受付時刻を変更しない

#### Scenario: 取得中の再通知
- **WHEN** 送信者が`running`のjobを持つ未保存sourceを有効なフォームで再通知する
- **THEN** リレーはHTTP 202を返す
- **AND** リレーは取得中のjobを継続する
- **AND** リレーは後続jobや並列取得を追加しない

#### Scenario: 同時に届く初回通知
- **WHEN** 複数の送信者がjobもスナップショットもない同じsourceを同時に通知し、キューに空きがある
- **THEN** リレーは一件のjobだけを永続化する
- **AND** リレーは全通知へHTTP 202を返す

#### Scenario: 集約後の正常処理
- **WHEN** リレーが重複通知を集約したjobを処理し、中断や一時障害なしで原本検証に成功する
- **THEN** リレーは原本を一回だけ取得する
- **AND** リレーは資格検証に成功したスナップショットを一件だけ保存する
- **AND** リレーは同じ通知群の後続jobを残さない

#### Scenario: 再試行待ちの再通知
- **WHEN** 送信者が一時障害後の再試行待ちjobを持つsourceを再通知する
- **THEN** リレーはHTTP 202を返す
- **AND** リレーは既存の試行回数と再試行予定を保持する
- **AND** リレーは再通知によって取得を前倒ししない

#### Scenario: 終端失敗後の新しい通知
- **WHEN** 送信者が`failed`の履歴だけを持つ未保存sourceを通知し、キューに空きがある
- **THEN** リレーは新しいjobを一件永続化してHTTP 202を返す
- **AND** リレーは過去の失敗を処理待ちjobとして扱わない
- **AND** リレーは新しいjobの試行回数をゼロから数える

#### Scenario: 保持期限後の新しい通知
- **WHEN** 送信者が期限削除後に処理待ちjobもスナップショットもないsourceを通知し、キューに空きがある
- **THEN** リレーは新しいjobを一件永続化してHTTP 202を返す
- **AND** リレーは削除済みjobを根拠として成功応答を返さない

## MODIFIED Requirements

### Requirement: 公開Webmention受付
リレーは認証や手動承認なしでWebmentionを受け付ける SHALL。リレーはHTTP POSTのフォームからsourceとtargetを読み、受付対象のtargetと異なるsourceを要求する SHALL。リレーは受付成功と掲載成功を区別する SHALL。リレーは、新しいjobの永続化または既存の処理待ちjobの永続保存を確認した場合だけHTTP 202を返す SHALL。リレーは、処理待ち重複の応答で既存通知の処理継続を示し、新規jobの追加や掲載成功を示さない SHALL。

#### Scenario: 未保存sourceの正常な通知
- **WHEN** 送信者がスナップショットも処理待ちjobもない有効なsourceと設定済みのtargetを送り、キューに空きがある
- **THEN** リレーは処理を永続キューに登録してHTTP 202を返す
- **AND** リレーは検証前の投稿をfeedへ掲載しない

#### Scenario: 不正な通知
- **WHEN** sourceまたはtargetが欠落する、同値になる、またはtargetが受付対象ではない
- **THEN** リレーはHTTP 400を返す
- **AND** リレーは外部取得とキュー登録を行わない

#### Scenario: 処理待ちsourceの不正なtarget
- **WHEN** 送信者が処理待ちsourceと受付対象外のtargetを送る
- **THEN** リレーはHTTP 400を返す
- **AND** リレーは既存jobのtargetを変更しない

#### Scenario: 永続保存の確認失敗
- **WHEN** リレーが新規jobの永続化または既存の処理待ちjobの永続保存の確認に失敗する
- **THEN** リレーはHTTP 503を返す
- **AND** リレーはメモリ上の重複情報だけでHTTP 202を返さない

#### Scenario: 保存後かつjob完了前の再通知
- **WHEN** 送信者がスナップショットと処理待ちjobの両方を持つsourceを有効なフォームで再通知する
- **THEN** リレーは保存済みsourceとしてHTTP 200を返す
- **AND** リレーは新しいjobや外部取得を追加しない
- **AND** リレーは最初のスナップショットを保持する

### Requirement: 受付資源の制限
リレーはリクエストサイズと未処理通知数に有限の上限を設ける SHALL。リレーは新規通知の上限超過をHTTPエラーで通知し、受付済みと偽らない SHALL。リレーは、容量判定より先に保存済みsourceと処理待ちsourceを判定する SHALL。リレーは、スナップショットも処理待ちjobもないsourceだけを追加のキュー容量の対象とする SHALL。

#### Scenario: キューの容量超過
- **WHEN** 永続キューが容量に達し、スナップショットも処理待ちjobもないsourceの通知が届く
- **THEN** リレーはHTTP 503を返す
- **AND** リレーは保存していない通知を受付済みと偽らない

#### Scenario: 満杯時の保存済みsource
- **WHEN** 永続キューが容量に達し、有効なフォームが保存済みsourceを通知する
- **THEN** リレーはHTTP 200を返し、追加のキュー容量を使わない

#### Scenario: 満杯時の処理待ちsource
- **WHEN** 永続キューが容量に達し、有効なフォームが`pending`または`running`のjobを持つ未保存sourceを通知する
- **THEN** リレーはHTTP 202を返す
- **AND** リレーは追加のキュー容量を使わない
- **AND** リレーは既存jobを変更しない

#### Scenario: 満杯直前の処理待ちsource
- **WHEN** 永続キューの空きが一枠あり、処理待ちsourceの再通知に続いて異なる新規sourceの通知が届く
- **THEN** リレーは両通知へHTTP 202を返す
- **AND** リレーは新規sourceのjobだけを一件追加する

#### Scenario: 終端失敗後の通知と満杯のキュー
- **WHEN** 永続キューが容量に達し、`failed`の履歴だけを持つ未保存sourceの通知が届く
- **THEN** リレーはHTTP 503を返す
- **AND** リレーは失敗履歴を根拠として受付済みと偽らない
