# Spec Delta

## Purpose

リレーはページの所属先と投稿を曖昧なく対応させる。保存データはsource URLごとの最初の検証成功スナップショットとし、microformats2の意味構造を保持してHTML表現を持たない。将来の表示機能はAtom専用モデルではなく、この構造化データを利用する。

## ADDED Requirements

### Requirement: UUID URN限定の所属先
リレーはページのrel=collectionから所属先を読む SHALL。初版のリレーは異なる所属先を一つだけ要求する SHALL。リレーはurn:uuid接頭辞とハイフン付きUUIDの完全な文字列だけを受け付け、不正な入力を補正しない SHALL。

#### Scenario: 有効なUUID URN
- **WHEN** ページが `urn:uuid:550e8400-e29b-41d4-a716-446655440000` を所属先として宣言する
- **THEN** リレーはそのUUIDのcollectionへ投稿を分類する

#### Scenario: 不正な所属先
- **WHEN** ページが裸のUUID、HTTP URL、波括弧形式、ハイフンなし形式、前後の空白、フラグメント付きURNを宣言する
- **THEN** リレーはそのページを投稿資格なしと判定する
- **AND** リレーは識別子を補正しない

#### Scenario: 複数の所属先
- **WHEN** ページが異なるcollectionを二つ以上宣言する
- **THEN** リレーはそのページを投稿資格なしと判定する
- **AND** リレーは最初の所属先を勝手に選ばない

#### Scenario: 同じ所属先の重複表記
- **WHEN** ページが同じUUIDのcollectionを複数回宣言する
- **THEN** リレーは同じ所属先の重複を一つと数える

### Requirement: 一意なh-entry
初版のリレーはページ内のh-entryを一つだけ要求する SHALL。リレーはh-feedの子項目を含めてh-entryを数える SHALL。リレーはh-entryがゼロまたは複数のページから投稿を推測しない SHALL。

#### Scenario: 投稿とプロフィールの共存
- **WHEN** ページがh-entryを一つ持ち、そのauthorが入れ子のh-cardを含む
- **THEN** リレーはh-entryだけを投稿として扱う
- **AND** リレーはh-cardを投稿の入れ子データとして保持する

#### Scenario: 投稿の欠落または複数投稿
- **WHEN** ページがh-entryを含まない、またはh-feed内を含めて複数のh-entryを含む
- **THEN** リレーはそのページを投稿資格なしと判定する

### Requirement: HTML表現を持たないmf2
リレーはmf2のtype、properties、children、入れ子の値を保持する SHALL。リレーはコンテンツのvalueをプレーンテキストとして保持する SHALL。リレーは入れ子を含むhtml表現を永続化前に除去し、取得HTMLと解析DOMを保存しない SHALL。

#### Scenario: HTML付きコンテンツ
- **WHEN** パーサーがコンテンツのhtmlとvalueを返す
- **THEN** 保存したmf2はvalueを保持する
- **AND** 保存したmf2はhtmlを含まない

#### Scenario: 入れ子のHTML表現
- **WHEN** 著者h-cardなどの入れ子がHTML表現を含む
- **THEN** リレーは入れ子の構造を保持する
- **AND** リレーは入れ子のHTML表現も除去する

#### Scenario: 保存後の読み戻し
- **WHEN** リレーが保存済み投稿を読み戻す
- **THEN** 投稿はHTMLを除去したmf2と同じ構造とテキストを持つ

### Requirement: UUIDの同一性
リレーは有効なUUIDの同一性をUUID値で判定する SHALL。リレーは入力のUUID部分の大文字と小文字を区別しない SHALL。リレーは公開feed idに小文字の正規UUID URNを使う SHALL。

#### Scenario: UUID部分の大小文字
- **WHEN** 異なる投稿がUUID部分の大小文字だけが異なる有効なURNを宣言する
- **THEN** リレーは両方を同じcollectionへ分類する
- **AND** 両投稿のfeedは同じ小文字UUID URNをidに使う

### Requirement: sourceごとの不変な保存
リレーはsource URL単独の一意制約で投稿を一件だけ保存する SHALL。リレーは最初の検証成功時のmf2、collection、target、最終取得URL、初回保存日時を固定する SHALL。リレーは競合する挿入と再通知で既存投稿を上書きしない SHALL。

#### Scenario: 初回保存の競合
- **WHEN** 同じsourceの初回保存が競合する
- **THEN** 投稿は一件だけ保存される
- **AND** 最初に保存成功したスナップショットの内容と所属を保持する

#### Scenario: 同じsourceの所属変更
- **WHEN** 保存後に原本のcollectionが変わり、同じsourceが再通知される
- **THEN** 保存済みのcollectionとmf2は変わらない
- **AND** 新しいcollectionへの移動または複製を行わない
