# indie-mezzanine

個人サイトのページを、テーマの決まっていないフィードに追加できる公開リレーです。AtomとJF2 Feedで配信します。著者のオプトイン式です。

## 動機

[中2階](https://nawashiro.dev/posts/20261005-mezzanine)のIndieWeb実装です。

- フィードに共通の話題がなんとなくある。明示はされない。空気は読んでもいいし、無視してもいい。
- 小さな空間だ。SNSのような大海に放り出されない。
- 誰でも始められる。識別子はただのランダムな文字列だ。

## 投稿

ページを[Microformat2](https://developer.mozilla.org/ja/docs/Web/HTML/Guides/Microformats)でマークアップしてください。サーバーは1ページに1件の`h-entry`を読みます。

1. フィードはUUIDで区別します。友達から貰ってくるか、生成してください。例 `bbe44bfe-3a9c-410e-ad37-f1bf7402bce4`
2. `rel="collection"`リンクでUUIDを指定してください。例 `<link rel="collection" href="urn:uuid:bbe44bfe-3a9c-410e-ad37-f1bf7402bce4">`
3. [このページ](/)へのリンクを置いてください。

公開したら、リンク宛に[Webmention](https://indieweb.org/Webmention)を送信してください。受け付けると`HTTP 202`を返します。最初に検証が成功した内容だけを保存します。更新には対応していません。

[mf2を含むマークアップ例はこちら。](https://tangled.org/nawashiro.dev/indie-mezzanine/blob/main/examples/post.html)

## 閲覧

著者が`rel="collection"`で指定したUUIDを購読できます。

例:

- Atom: [/collections/bbe44bfe-3a9c-410e-ad37-f1bf7402bce4.atom](/collections/bbe44bfe-3a9c-410e-ad37-f1bf7402bce4.atom)
- [JF2 Feed](https://jf2.spec.indieweb.org/): [/collections/bbe44bfe-3a9c-410e-ad37-f1bf7402bce4.jf2](/collections/bbe44bfe-3a9c-410e-ad37-f1bf7402bce4.jf2)

## テクニック

複数のリレーに送って冗長化することもできます。IDは同じなので、必要ならJF2を拾い集めてマージしてもよいです。

```html
<link rel="collection" href="urn:uuid:bbe44bfe-3a9c-410e-ad37-f1bf7402bce4" />
<a href="https://alice.example.com/"></a>
<a href="https://bob.example.com/"></a>
```

## リンク

- [ソースコード - Tangled](https://tangled.org/nawashiro.dev/indie-mezzanine)
