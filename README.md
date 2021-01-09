# 🐐 yagss

`yagss` is short for *yet another generator of static sites*. `yagss` supports blogs and non-blogs. It uses [Jinja](https://jinja.palletsprojects.com/en/2.11.x/) style templates via [pongo2](https://github.com/flosch/pongo2), supports markdown, RSS feed generation, cache-busting of static assets, and minifies output by default. Unlike [Jekyll](https://jekyllrb.com/) and [Hugo](https://gohugo.io/), there are no themes--just HTML templates and CSS, which you fully control.

`yagss` is intended help make simple websites where all you really need is some HTML, CSS, and maybe a bit of JavaScript. See the quickstart and documentation below for more information.

## Usage

```
Usage:
  yagss [command]

Available Commands:
  build       Build the current yagss site
  help        Help about any command
  new         Create a new yagss site
  serve       Serve the current yagss site and auto build when files change

Flags:
  -h, --help   help for yagss

Use "yagss [command] --help" for more information about a command.
```

## Quickstart

Create a new site.

```
$ yagss new demo
Creaing new yagss project in "demo"
==> Creating "demo" directory
==> Creating "demo/config.toml"
==> Creating "demo/posts" directory
...
==> Creating "demo/build" directory
DONE
$ cd demo
$ yagss serve
----> Initial build
[builder] Starting build...
[builder] ==> Processing "public/favicon.ico"
[builder] ==> Processing "public/styles.css"
[builder] ==> Processing "posts/first-post.md"
[builder] ==> Processing "posts/forth-post.md"
[builder] ==> Processing "posts/second-post.md"
[builder] ==> Processing "posts/third-post.md"
[builder] ==> Processing "pages/about.md"
[builder] ==> Processing "pages/index.html"
[builder] ==> Processing "rss.xml"
[builder] Processed 9 files in 7.760285ms
----> Starting server and watcher
[watcher] Watching "pages" directory
[watcher] Watching "posts" directory
[watcher] Watching "public" directory
[watcher] Watching "templates" directory
[server] Listening on port :3000
# Now browse to http://localhost:3000
```

I recommend checking out the files, changing things, and seeing what happens.

## Documentation

### Directory Structure

Here's the `yagss` directory structure and some information about each directory's role.

```
$ tree
.
├── build
├── config.toml
│   # config.toml allows for controling various build settings.
├── pages
│   # The pages directory contains pages that can be in markdown or
│   # html format. Pages can also use template directives and they
│   # are compiled to the site's output directory.
│   ├── about.md
│   └── index.html
├── posts
│   # The posts directory contains markdown blog posts. Its usage is  
│   # optional.
│   ├── first-post.md
│   ├── forth-post.md
│   ├── second-post.md
│   └── third-post.md
├── public
│   # Files and sub-directories in this directory are moved to the root of  
│   # the output directory. Files that end with .html, .css, .js, .jsx,
│   # .svg, .xml, or .json are automatically minified.
│   ├── favicon.ico
│   └── styles.css
└── templates
    # The templates directory contains templates and partials. Unlike
    # pages, they are not compiled on their own. In other words,
    # if a "templates/index.html" file exists, but no file in "pages"
    # refers to it, then it will not be included in the site's built
    # output. In contrast, "pages/index.html" will be included.
    ├── base.html
    ├── page.html
    ├── pagination.html
    └── post.html

5 directories, 13 files
```

### Building

Build the site with `yagss build`. By default, generated files will be placed in the `build/` directory. This can be changed by editing the `directories.output` setting in `config.toml`.

```
$ yagss build
Starting build...
==> Processing "public/favicon.ico"
==> Processing "public/styles.css"
==> Processing "posts/first-post.md"
==> Processing "posts/forth-post.md"
==> Processing "posts/second-post.md"
==> Processing "posts/third-post.md"
==> Processing "pages/about.md"
==> Processing "pages/index.html"
==> Processing "rss.xml"
Processed 9 files in 15.806605ms
```

Let's look at the generated files. Note that they are minified and that CSS assets contain hashes in their names. This makes cache-busting the default behavior. By default, hashes are added to `.js` and `.css` files. This can be changed by editing the `build.hash` setting in `config.toml`.

```
$ tree build
build/
├── about.html
├── favicon.ico
├── index.html
├── page2
│   └── index.html
├── posts
│   ├── first-post.html
│   ├── forth-post.html
│   ├── second-post.html
│   └── third-post.html
├── rss.xml
└── styles.df1b98dd.css
```

### Static Asset Handling

Assets must be referenced in templates and markdown files with the `assets` object and `key` filter. `assets` is a map of source-paths to output-paths, where its keys are source-paths of all files in the `public` directory. `assets` is made available to every template and markdown file.

```
$ cat templates/base.html | grep assets
    <link rel="icon" href="{{ assets|key:'favicon.ico' }}" />
    <link rel="stylesheet" href="{{ assets|key:'styles.css' }}" />
```

### Pages

Pages must be placed in the `directories.pages` directory and can be nested. The directory tree is preserved when building. Posts can be in markdown or HTML format and can use template directives. If pages are in markdown format, the `defaults.pageTemplate` is used to render the page. This can be overridden by specifying a `template` key in the markdown front-matter whose value is a path to a template in the `directories.templates` directory. Here's an example.

```
---
title: About
description: Optional description
template: "my-special-template.html" # Optional template override.
---
I am me--or am I?
```

And here's an example template:

```
{% extends 'base.html' %}
{% block content %}
  <img src="{{ assets|key:'images/me.jpg' }}" alt="special image in this template">
  <h2>{{ title }}</h2>
  {{ content|safe }}
{% endblock %}
```

Note that `content`, which is the rendered markdown content, uses the `safe` filter. This is important because otherwise the rendered markdown would be escaped.

##### Template Parameters for Pages

| Field | Type | Comment |
| ----- | ---- | ------- |
| pageTitle | String | The title of the page intended for use in the `<title>` tag. |
| pageDescription | String | The description of the page intended for use in the description `<meta>` tag. |
| siteURL | String | The base URL of the site. |
| assets | Map | A map of source-paths to output-paths for all files in the `directories.public` directory. |
| content | String | Optional. Rendered markdown from markdown file. |
| title | String | Optional. Passed from markdown file. |

### Blogging

Blog posts must be placed in the `posts` directory (This can be changed by editing the `directories.posts` setting). Each post must be a markdown file with front-matter that specifies a title and date. For example:

```
---
title: First post!
date: 2021-01-01
description: Optional description
---
![fireworks]({{ assets|key:'fireworks.jpg' }})

Hello world.
```

Note that the `assets` object is made available to posts as well.

Blog posts are rendered using the `defaults.postTemplate` file. A destructured post object is passed to this template. Here's an example of a `defaults.postTemplate` that renders the given fields. Note that `content` uses the `safe` filter. This is important because otherwise the rendered markdown would be escaped.

```
{% extends 'base.html' %}
{% block content %}
  <h2>{{ title }}</h2>
  <time datetime="{{ date }}">{{ date|date:"2 Jan 2006" }}</time>
  {{ content|safe }}
{% endblock %}
```

##### Post Object

| Field | Type | Comment |
| ----- | ---- | ------- |
| Title | String | Required. |
| Description | String | Optional. |
| Date | time.Time | Required. |
| Content | String | Required. The rendered markdown. |
| Path | String | Required. The relative URL of the post. |
| URL | String | Required. The absolute URL of the post. |


#### Posts Index

The `build.postsIndex` template file is used to generate a paginated list of all blog posts. A `posts` object is passed to this template that contains all of the posts for that page. `next` and `prev` strings are also passed to the `build.postsIndex` template in order to support pagination. Page size is determined by the `defaults.postsPerPage` setting. Here's an example posts index template:

```
{% extends 'base.html' %}
{% block content %}
  <!-- Iterate through all posts -->
  {% for post in posts %}
    <article>
      <h2>
        <a href="{{ post.Path }}">{{ post.Title }}</a>
      </h2>
      <time datetime="{{ post.Date }}">{{ post.Date|date:"2 Jan 2006" }}</time>
    </article>
  {% endfor %}

  <!-- Add links to next and prev pages, if there are any -->
  {% if prev %}
    <span>
      <a href="{{ prev }}">Newer</a>
    </span>
  {% endif %}

  {% if next %}
    <span>
      <a href="{{ next }}">Older</a>
    </span>
  {% endif %}
{% endblock %}
```

##### Template Parameters for Post Index

| Field | Type | Comment |
| ----- | ---- | ------- |
| pageTitle | String | The title of the page intended for use in the `<title>` tag. |
| pageDescription | String | The description of the page intended for use in the description `<meta>` tag. |
| siteURL | String | The base URL of the site. |
| assets | Map | A map of source-paths to output-paths for all files in the `directories.public` directory. |
| posts | []Post | An array of post objects. |
| next | String | Optional. Link to the next page of posts. |
| prev | String | Optional. Link to the previous page of posts. |

#### RSS

By default, an RSS feed is generated that uses the most recent `build.postsPerPage` posts. If `build.postsPerPage` is three, then the most recent three posts will be included in the resulting `rss.xml`. This can be disabled by making the `build.rss` setting `false` in `config.toml`.

