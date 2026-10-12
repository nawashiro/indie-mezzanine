# Spec Delta

## ADDED Requirements

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
