# Tasks

各グループは独立した検証単位とする。実装者はその単位の検証後にだけチェックを付ける。テストは実記事ではなく固定fixtureを使う。本変更は保存モデルを変えず、配信の出口と契約テストだけを増やす。

## 1. 射影規則とfixture

- [x] 1.1 保存済みmf2 JSONの形状（`type`/`properties`/`children`）を固定fixtureで確認し、design.mdの正規化規則をテストヘルパと期待JSONへ落とす — verify: fixtureが `Parse` を通り、規則どおりの期待JSONと一致し、`value`/`id`/`lang`/`html` が現れない。
- [x] 1.2 複数photo・入れ子h-card著者・children(h-cite)・identity property（`uid`/`url`）・u-email・`video`/`audio`・タグとアンパサンドを含むcontentの固定fixtureを追加する — verify: 解析後の保存JSONがfixtureどおり（HTML除去込み）で、射影テストの入力として再利用できる。
- [x] 1.3 空配列・除去後に空になるproperty・日時欠落の境界fixtureを追加する — verify: 各境界が期待どおり省略され、`email` だけが常に消える。

## 2. JF2生成とプロファイル準拠

- [x] 2.1 保存済みの並びからJF2 Feedを生成する関数を追加し、`type: "feed"` と `children` の `type: "entry"`、各entryの単一文字列 `uid` を出力する — verify: JSON解析テストで封筒と `uid` を確認する。
- [x] 2.2 メタデータ・本文・正規化を実装する — verify: `name` の欠落時source URL、`author` のcardと代替値、有効時のみの `published`、`updated` の初回保存日時への代替、`content.text` のみで `html` なし、`category` 配列、`photo` 先頭単一文字列、`video`/`audio` オブジェクト、入れ子 `value` なし、`email` なしを検査するテストが成功する。
- [x] 2.3 順序・上限・決定性を実装する — verify: entry.updated降順とentry id昇順、`FEED_LIMIT` 上限、同じ保存内容の再取得でバイト一致、再通知後も不変のテストが成功する。
- [x] 2.4 プロファイル準拠を確認する — verify: 必須項目（`type`/`children`/`uid`/`name`）、日時形式、単一文字列制約の検査が成功し、可能ならjf2.rocksのバリデータでfixtureを検証する。

## 3. 経路・HTTP契約・文書

- [x] 3.1 `GET /collections/{name}` に `.jf2` 分岐を追加し、`application/jf2feed+json` をcharsetなしで返す — verify: HTTPテストで200とContent-Type（charsetパラメータなし）を確認する。
- [x] 3.2 不正UUID・未知collection・`.atom`/`.jf2` 以外の接尾辞の404と、認証なしの取得を維持する — verify: 各ケースのHTTPテストが404または200を返す。
- [x] 3.3 READMEへ `.jf2` の購読URLとメディアタイプ、ビューワーとCORSが対象外であることを追記する — verify: 文書どおりのURLがハンドラで解決し、200とJF2 Feedを返す。

## 4. 形式間の一貫性

- [x] 4.1 同一fixtureでAtomとJF2のentry集合・順序・id/uid・日時文字列の一致を検証する — verify: 契約テストが両形式で同じ集合・順序・識別子・日時になる。
- [x] 4.2 JF2追加で既存Atomの応答が変わらないことを確認する — verify: 既存の契約テスト（`TestExampleSubmissionAndFeed` 等）がそのまま成功する。

## 5. 統合確認

- [x] 5.1 受付から保存、両形式の配信までのE2Eを実行する — verify: 同じ固定sourceで両形式の集合・順序・識別子・日時が一致し、再通知後も両形式が不変となる。
- [x] 5.2 静的検査とOpenSpec検証を実行する — verify: `go vet ./...`、`go test -race ./...`、`openspec validate publish-collection-jf2 --strict` が成功する。実行できない検査は未完了として報告する。

## Workflow follow-up

- レビュー要件を満たした後に変更をアーカイブする。
- アーカイブ後、`openspec/specs/collection-feeds/spec.md` にjf2の要求とPurposeが意図どおり統合されたことを確認する。
