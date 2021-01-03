# yasst

`yasst` is short for yet another static site.

```
# Take a directory tree like this one.
$ tree
.
├── pages
│   ├── 404.html.tpl
│   ├── about.md
│   └── index.html.tpl
├── posts
│   └── first-post.md
├── public
│   ├── scripts.js
│   ├── styles.css
│   └── my-image.jpg
└── templates
    ├── base.html.tpl
    ├── page.html.tpl
    └── post.html.tpl

# Run yasst.
$ yasst --output ./static
==> Processing "public/scripts.js" --> DONE
==> Processing "public/styles.css" --> DONE
==> Processing "public/my-image.jpg" --> DONE
==> Processing "posts/first-post.md" --> DONE
==> Processing "pages/404.html.tpl" --> DONE
==> Processing "pages/about.md" --> DONE
==> Processing "pages/index.html.tpl" --> DONE
Processed 7 files in 4.15023ms

# Here's the result. Note that files in public were automatically minified and hashed.
$ tree static
static/
├── index.html
├── about.html
├── 404.html
├── rss.xml
├── posts
│   └── first-post.html
├── scripts-78hdyfuis.min.js
├── styles-gyhuij9.min.js
└── my-image.jpg

```
