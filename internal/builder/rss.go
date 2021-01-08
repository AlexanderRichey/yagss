package builder

const rssT = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>{{ title }}</title>
    <link>{{ url }}</link>
    <language>en-us</language>
    <description>{{ description }}</description>
    <pubDate>{{ date }}</pubDate>
    <lastBuildDate>{{ date }}</lastBuildDate>
    {% for post in posts %}
    <item>
      <title>{{ post.Title }}</title>
      <link>{{ post.URL }}</link>
      <pubDate>{{ post.Date }}</pubDate>
      <description>{{ post.Content|striptags }}</description>
    </item>
    {% endfor %}
  </channel>
</rss>`
