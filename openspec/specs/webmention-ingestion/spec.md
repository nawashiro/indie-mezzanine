# webmention-ingestion Specification

## Purpose

公開リレーはWebmentionを受け付け、取得とリンク検証に成功した原本をスナップショットとして掲載する。リレーはsource URLごとの最初の検証成功結果を固定し、再通知による上書き、所属移動、撤回を行わない。

## Requirements

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

### Requirement: HTTP(S)ルートtargetの限定的な同値性
リレーはHTTP(S)ルートURLの空パスと `/` を同値とする SHALL。リレーは設定URL、通知target、解決済み原本リンクに同じ規則を適用する SHALL。リレーは追加する同値規則をこのパス差だけに限定する SHALL。リレーは既存の設定受理範囲とURL解析・参照解決・直列化の動作を維持する SHALL。リレーは比較時にも設定ルートと異なる通知targetを拒否する SHALL。

#### Scenario: 設定と通知の両ルート表記
- **WHEN** 運用者が `https://relay.example` または `https://relay.example/` を設定する
- **AND** 送信者が別の未保存sourceと、どちらかのルート表記のtargetを送る
- **THEN** リレーは四つの組合せをHTTP 202で受け付ける
- **AND** リレーは検証前の投稿をfeedへ掲載しない

#### Scenario: HTTP設定の両ルート表記
- **WHEN** 運用者が `http://relay.example` または `http://relay.example/` を設定する
- **AND** 送信者が別の未保存sourceと、同じHTTPルートのどちらかのtargetを送る
- **THEN** リレーは四つの組合せをHTTP 202で受け付ける

#### Scenario: 解決済みリンクの両ルート表記
- **WHEN** 安全に取得した原本が有効な単一collectionと一意なh-entryを持つ
- **AND** 原本リンクの解決結果と受付targetが、同じルートの空パスと `/` だけで異なる
- **THEN** リレーはそのリンクをtargetリンクと判定する
- **AND** リレーは検証結果を初回スナップショットとして掲載する

#### Scenario: 相対参照と文書の基底URL
- **WHEN** 原本の相対リンクが最終取得URLと文書の基底URLによって受付ルートへ解決する
- **THEN** リレーは解決後に空パスと `/` の同値性を判定する
- **AND** リレーはcollection宣言だけをtargetリンクと見なさない

#### Scenario: 受付対象ではないtarget
- **WHEN** targetが別ホスト、別スキーム、異なるポート表記、または非ルートパスを持つ
- **THEN** リレーはHTTP 400を返す
- **AND** リレーはキュー登録と外部取得を行わない

#### Scenario: targetのqueryと非空fragment
- **WHEN** targetがquery、空の `?`、空でないfragment、または認証情報を含む
- **THEN** リレーはHTTP 400を返す
- **AND** リレーはそれらを除去して受け付けない

#### Scenario: 対象ではない原本リンク
- **WHEN** 原本リンクの解決結果が受付ルートと異なるホスト、スキーム、非ルートパス、query、または空でないfragmentを保持する
- **AND** 原本が他の有効なtargetリンクを持たない
- **THEN** リレーは当該通知を検証失敗とする
- **AND** リレーは投稿を掲載しない

#### Scenario: 他のURL同値性を追加しない
- **WHEN** URL解析後も通知targetが設定URLとホスト表記、明示ポート、またはルート以外のパス表記で異なる
- **THEN** リレーはルートパスの同値規則だけではそのtargetを受け付けない

#### Scenario: 既存設定の受理範囲
- **WHEN** 運用者が変更前の設定検証を通過する公開URLを指定する
- **THEN** リレーは同じ設定を受理する
- **AND** リレーは本変更で設定の禁止条件を追加しない

### Requirement: ルート自己通知の拒否
リレーは受付ルートと同値なsourceを拒否する SHALL。リレーは自己判定に限りsourceのfragmentを除去し、空パスと `/` を同値とする SHALL。リレーはこの比較によってsourceの保存表記を変更しない SHALL。

#### Scenario: 自己sourceの両ルート表記
- **WHEN** 設定URLとtargetが同じ受付ルートのどちらかの表記を使う
- **AND** sourceがそのルートの空パスまたは `/` を使う
- **THEN** リレーは全組合せでHTTP 400を返す
- **AND** リレーはキュー登録と外部取得を行わない

#### Scenario: fragment付きの自己source
- **WHEN** sourceが受付ルートのどちらかの表記に `#section` または空の `#` を付ける
- **THEN** リレーは設定とtargetの両表記に対してHTTP 400を返す
- **AND** リレーは自己判定からfragmentを除外する

#### Scenario: 自己判定を任意パスへ拡張しない
- **WHEN** sourceが受付ホストの `/post` または `/post/` を指す
- **AND** sourceが既存の取得URL検査を通過する
- **THEN** リレーはルート自己通知の規則だけではsourceを拒否しない

### Requirement: ルート表記変更時の保存互換性
リレーは任意sourceの入力文字列を投稿識別子として維持する SHALL。リレーは既存jobの両ルートtarget表記を検証する SHALL。リレーはこの変更のために既存jobのtargetや既存投稿を一括書換えしない SHALL。リレーは既存の再通知と有限再試行の規則を維持する SHALL。

#### Scenario: 未処理jobと中断jobの再開
- **WHEN** データストアが空パスと `/` のtargetを持つ未掲載の受付済みjobを保持する
- **AND** リレーが同じ公開ルートのどちらかの設定表記で再起動する
- **THEN** リレーは通常の復旧規則でjobを処理する
- **AND** リレーは原本リンクの両ルート表記を検証する
- **AND** リレーは既存jobのtarget表記を一括変更しない

#### Scenario: 保存済み投稿の保持
- **WHEN** 同じ公開ルートの設定表記を変更してリレーが再起動する
- **AND** 保存済みsourceがどちらかの有効なルートtargetで再通知する
- **THEN** リレーはHTTP 200を返す
- **AND** リレーはsourceを再取得せず、投稿と識別子を変更しない

#### Scenario: 任意sourceの表記を識別子として保持
- **WHEN** 別ホストの `https://source.example` と `https://source.example/` がそれぞれ初回検証に成功する
- **THEN** リレーは各入力文字列を別のsource識別子として保持する
- **AND** リレーは `/post` と `/post/` のsource識別子も統合しない

#### Scenario: 過去の失敗後の再通知
- **WHEN** 過去のjobが検証失敗で終了し、同じ未保存sourceの新しい通知が検証に成功する
- **THEN** リレーは新しい成功結果を最初のスナップショットとして保存する
- **AND** リレーは過去の失敗だけを理由に永久拒否しない

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
