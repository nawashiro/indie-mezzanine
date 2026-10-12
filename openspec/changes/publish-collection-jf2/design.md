# Design

## Context

動機はproposal.mdを参照する。設計を決める現状と制約:

- リレーは保存済みmf2（全階層のHTMLを除去したもの）を正本とする。原本を再取得せず、再通知で保存済み投稿を変えない（`internal/relay/parse.go`、`internal/relay/store.go`）。
- 既存の出口はAtomのみ。本文はtext、著者写真・photo・category・summaryは出力しない。出力層は `gorilla/feeds` のAtom専用型を直接組み立てる（`internal/relay/feed.go`）。
- 初回設計はNon-Goalsで「mf2の段落を独自HTMLレンダラーで復元しない」、リスク節で「GUIとURL表示は別変更で検査する」としている。本変更はこの方針を維持し、HTMLを生成せず構造だけを出す。
- 本文の段落境界は取り込み時に完全には保たれない。これは出力形式では解決できない。
- 公開契約の候補はJF2 Feedプロファイル（W3C Note jf2 2018-01-10、IndieWeb living standard）。トップレベル `type: "feed"`、`children` 配列の `type: "entry"`、`uid` 必須、メディアタイプ `application/jf2feed+json` を定義する。
- 既存経路は `GET /collections/{name}` で `.atom` 接尾辞を必須にし、不正UUIDと未知collectionに404を返す（`internal/relay/server.go`）。上限は `FEED_LIMIT`（既定100）。

## Goals / Non-Goals

**Goals:**
- 保存済みスナップショットの構造を、JF2 Feedプロファイル準拠のJSONとして決定的に公開する。
- 既存Atomの出力と不変条件を変えない。
- 射影規則を仕様とテストに固定し、外部プロファイルの変化へ無言で追従しない。

**Non-Goals:**
- ビューワー本体、配信、ホスティング。
- CORS許可と発見経路（`rel="alternate"`等）。ビューワーの置き場所が決まった時点の別変更とする。
- HTMLマークアップの生成とサニタイズ層、段落境界の再構成、スナップショットとスキーマの変更。
- 画像の取得・プロキシ。写真はURLデータとしてのみ扱う。
- JSON Feed等の別語彙、Microsubサーバー実装、複数collectionの集約。

## Decisions

### 1. 契約はJF2 Feedプロファイル。JSON Feedや生mf2 JSONではない

保存正本がmf2であり、辞書を新設せずに済む。プロファイルが封筒（`feed` / `children` / `entry` / `uid`）とメディアタイプを定義済みで、検証器もある。生mf2 JSONは配列中心で読む側にmf2知識を要求し、JSON Feedは別語彙への写像を維持する負担が増える。AtomにHTMLを合成する案は、Non-Goalsの独自レンダラー復元に当たり、段落の捏造とサニタイズ面を持ち込むため採らない。

### 2. 経路とメディアタイプは `.atom` に並置する

`GET /collections/{uuid}.jf2` を追加し、`application/jf2feed+json` をcharsetパラメータなしで返す（プロファイルのエンコーディング規定に合わせる）。コンテンツネゴシエーションは導入せず接尾辞で分岐し、既存の404挙動を共有する。見えにくい分岐を増やさない。

### 3. 射影は機械的な正規化とし、propertyを列挙しない

ビューワーの要望のたびにリレー変更が要るのを避け、保存済みmf2の全propertyを次の規則で正規化して通す。

| 入力 | 出力 |
| --- | --- |
| identity (`uid`, `url`) | source URLで固定。保存済みmf2の値は使わない |
| `name` | mf2の名称。欠落時はsource URL |
| `content` | `{"text": value}`。`html`は出さない。contentが無ければ出力しない |
| `author`（入れ子h-card） | `card`へ再帰正規化。名称の欠落時は `投稿者不明` |
| `photo` | 先頭の単一文字列。altと残りは出さない |
| `video`, `audio` | `{"url": ...}`（複数時はオブジェクト配列） |
| `category` | 文字列の配列（常に配列） |
| `published`, `updated` | Atomと同じ文字列。`published`は有効時のみ、`updated`は欠落時に初回保存日時へ代替 |
| その他property | 型を保って通す。単一要素の配列は単一値へ畳む |
| `type` | `h-` 接頭辞を除去 |
| 入れ子オブジェクト | `properties` と `children` を再帰正規化し、`value`/`id`/`lang` は出さない |
| `email` | 出力しない（連絡先denylist） |
| 空配列・除去後に空になったproperty | 出力しない |

feedメタデータはAtomと対応させる。`name` はcollectionの正規URN、`url` はそのリレーのJF2 FeedのURL、`author` は `indie-mezzanine` のcardとする。

### 4. 連絡先だけを除き、他は素通しする

素通しの根拠は著者のオプトインと元ページの公開にある。ただしfeedは収集の的になりやすく、h-cardの `email` は元ページが公開していても載せない。他の連絡先プロパティは必要になった時点で別変更としてdenylistへ加える。

### 5. Atomの生成経路は触らない

既存 `Atom` は変更せず、JF2は同じヘルパー（`properties` / `first` / `text` / `parseDate` / `entryTime`）と保存済みの並びを使う別関数として追加する。中間表現の共有リファクタはしない。Atomの応答を変えないという要求に対し、共有化はリスクに見合わない。形式間の一致は契約テストで検証する。

### 6. 決定的出力と検証

`encoding/json` はmapキーをソートするため出力は安定する。テストは再取得のバイト一致に加え、JSON解析後の値とプロファイルMUST（`type` / `children` / `uid` / 日時形式 / 単一文字列制約）を確認する。プロファイル準拠は可能ならjf2.rocksで手動検証する。

## Risks / Trade-offs

- [プロファイルはEditor's Draftを含むliving standard] → 準拠の基準を本リポジトリの仕様とテストに固定し、外部仕様の変化で出力が無言に変わらないようにする。
- [素通し射影が想定外のpropertyを公開する] → 根拠をオプトインと元ページ公開に置き、`email` をdenylistにする。追加の非公開項目は別変更で扱う。
- [AtomとJF2の食い違い] → 同じ保存スナップショットと同じ並びを使い、集合・順序・識別子・日時文字列の一致を契約テストで固定する。
- [画像URLの表示で読者IPが原本サイトへ出る] → リレーは画像を取得・プロキシしない。mixed content等も含めビューワー側の判断とする。
- [`.jf2` 追加で既存Atomが変わる] → Atomの生成経路に触れず、既存契約テストで応答の維持を確認する。
- [本文の段落境界は既に失われている] → 本変更は構造の公開に限定し、段落の再構成は取り込み側の別変更とする。

## Migration Plan

スキーマと保存データは変わらないため移行は不要。配備はイメージ差し替えだけで `.jf2` 経路が増える。ロールバックは以前のイメージへ戻すと `.jf2` が消えるだけで、Atom応答と保存データは不変。
