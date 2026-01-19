# Static Site Generator CLI (Advanced)

Build a production-ready static site generator that transforms Markdown content into a complete website with templates, asset processing, and a development server.

## Technical Stack
- Python 3.11+
- Click for CLI framework
- Jinja2 for templating
- Markdown library with extensions (tables, fenced code, footnotes)
- Pygments for syntax highlighting
- PyYAML for frontmatter and config
- watchdog for file watching
- pytest for testing
- No external database

## Core Features

### Content Processing
- Parse Markdown files with YAML frontmatter
- Support for posts (blog entries with dates) and pages (static content)
- Automatic slug generation from filename or title
- Draft support (skip drafts in production builds)
- Scheduled publishing (future dates hidden until publish time)

### Frontmatter Schema
```yaml
---
title: "My Post Title"
date: 2024-01-15
updated: 2024-01-20
draft: false
template: post.html
tags: [python, tutorial]
category: programming
author: Jane Doe
description: "A brief summary for SEO and feeds"
featured_image: /images/hero.jpg
---
```

### Markdown Extensions
- Fenced code blocks with syntax highlighting (Pygments)
- Tables (GitHub-flavored)
- Footnotes
- Table of contents generation (`[TOC]`)
- Auto-linking URLs
- Strikethrough (`~~text~~`)
- Task lists (`- [ ] todo`)
- Image captions (`![alt](src "caption")`)
- Admonitions/callouts (`!!! note "Title"`)

## CLI Commands

### `ssg build`
Build the complete site.
- `--config <file>` (default: `config.yaml`)
- `--output <dir>` (default: `_site`)
- `--drafts` (include draft posts)
- `--future` (include future-dated posts)
- `--minify` (minify HTML/CSS/JS output)
- `--base-url <url>` (override base URL for builds)

### `ssg serve`
Development server with live reload.
- `--port <port>` (default: 8000)
- `--host <host>` (default: localhost)
- `--drafts` (include drafts)
- `--no-reload` (disable live reload)
- Watch for changes and rebuild automatically
- Inject live reload script into HTML

### `ssg new <type> <title>`
Create new content from templates.
- Types: `post`, `page`
- Creates file with frontmatter template
- Auto-generates filename from title and date
- `--template <name>` (use custom archetype)

### `ssg init <directory>`
Initialize new site structure.
- Creates directory structure
- Copies default theme
- Creates sample config.yaml
- Creates example content

### `ssg clean`
Remove build output directory.

### `ssg check`
Validate site configuration and content.
- Check for broken internal links
- Validate frontmatter schema
- Check for missing images
- Warn about draft/future content
- Validate template syntax

## Configuration (`config.yaml`)

```yaml
site:
  title: "My Awesome Blog"
  description: "A blog about things"
  author: "Jane Doe"
  url: "https://example.com"
  language: "en"

content:
  posts_dir: "content/posts"
  pages_dir: "content/pages"
  drafts_dir: "content/drafts"

build:
  output_dir: "_site"
  clean_before_build: true

theme:
  name: "default"
  custom_dir: "themes/mytheme"

templates:
  post: "post.html"
  page: "page.html"
  list: "list.html"
  home: "home.html"
  tag: "tag.html"
  category: "category.html"

permalinks:
  posts: "/:year/:month/:slug/"
  pages: "/:slug/"
  tags: "/tags/:slug/"
  categories: "/categories/:slug/"

pagination:
  enabled: true
  per_page: 10

assets:
  static_dir: "static"
  copy_patterns:
    - "images/**/*"
    - "fonts/**/*"
    - "favicon.ico"
  
css:
  minify: true
  bundle: true
  files:
    - "css/main.css"
    - "css/syntax.css"

js:
  minify: true
  bundle: true
  files:
    - "js/main.js"

markdown:
  extensions:
    - tables
    - fenced_code
    - footnotes
    - toc
    - smarty
  syntax_highlight: true
  syntax_theme: "monokai"

feed:
  enabled: true
  formats: [rss, atom, json]
  limit: 20

sitemap:
  enabled: true
  changefreq: weekly
  priority: 0.5

taxonomies:
  tags: true
  categories: true

plugins:
  - ssg_reading_time
  - ssg_related_posts
```

## Template System (Jinja2)

### Available Variables
- `site` - Site configuration object
- `page` - Current page/post object
- `content` - Rendered HTML content
- `posts` - All posts (sorted by date)
- `pages` - All pages
- `tags` - All tags with post counts
- `categories` - All categories with post counts

### Page Object Properties
- `title`, `date`, `updated`, `author`
- `url` - Permalink URL
- `content` - Rendered HTML
- `excerpt` - First paragraph or manual excerpt
- `reading_time` - Estimated minutes to read
- `word_count` - Total words
- `tags`, `category`
- `prev`, `next` - Adjacent posts
- `related` - Related posts by tags

### Built-in Filters
- `date(format)` - Format date (`{{ page.date | date("%B %d, %Y") }}`)
- `excerpt(words=50)` - Truncate to excerpt
- `reading_time` - "5 min read"
- `slugify` - Convert to URL slug
- `markdownify` - Render Markdown string
- `absolute_url` - Convert relative to absolute URL
- `asset_url` - Add cache-busting hash

### Template Inheritance
```html
{# base.html #}
<!DOCTYPE html>
<html>
<head>
  <title>{% block title %}{{ site.title }}{% endblock %}</title>
  {% block head %}{% endblock %}
</head>
<body>
  {% include "partials/nav.html" %}
  {% block content %}{% endblock %}
  {% include "partials/footer.html" %}
</body>
</html>

{# post.html #}
{% extends "base.html" %}
{% block title %}{{ page.title }} | {{ site.title }}{% endblock %}
{% block content %}
<article>
  <h1>{{ page.title }}</h1>
  <time>{{ page.date | date("%B %d, %Y") }}</time>
  {{ content }}
</article>
{% endblock %}
```

## Asset Pipeline

### CSS Processing
- Concatenate multiple CSS files (in order specified)
- Minify CSS (remove whitespace, comments)
- Generate source maps in dev mode
- Add content hash to filename for cache busting

### JavaScript Processing
- Concatenate JS files
- Minify with basic minification (remove whitespace, comments)
- Generate source maps in dev mode
- Content hash for cache busting

### Image Handling
- Copy images to output
- Generate responsive image srcsets (optional)
- Optimize images (optional, requires Pillow)

### Static Files
- Copy static directory contents to output root
- Preserve directory structure
- Pattern-based inclusion/exclusion

## Generated Files

### RSS/Atom/JSON Feed
- `/feed.xml` (RSS 2.0)
- `/atom.xml` (Atom 1.0)
- `/feed.json` (JSON Feed 1.1)
- Include recent N posts
- Full content or excerpt option

### Sitemap
- `/sitemap.xml`
- Include all pages and posts
- Set priority and changefreq per content type
- Exclude drafts and noindex pages

### Taxonomy Pages
- `/tags/` - List all tags
- `/tags/<tag>/` - Posts with tag
- `/categories/` - List all categories
- `/categories/<category>/` - Posts in category

### Pagination
- `/blog/` - First page
- `/blog/page/2/` - Subsequent pages
- Include prev/next links in template context

## Development Server

### Features
- HTTP server on configurable port
- Automatic rebuild on file changes
- Live reload via WebSocket or polling
- Show build errors in browser
- Serve 404 page for missing routes
- Handle clean URLs (try `/path/index.html`)

### Watch Behavior
- Watch content, templates, static, config
- Debounce rapid changes (100ms)
- Incremental rebuild when possible (content changes only)
- Full rebuild on template/config changes

## Error Handling
- Invalid frontmatter: error with filename and line
- Template syntax errors: show template name and line
- Missing template: error with suggestions
- Missing includes: error with include path
- Build errors: don't write partial output
- Dev server: show errors in browser overlay

## Performance
- Incremental builds: track file modification times
- Parallel Markdown processing
- Cache compiled templates
- Lazy load content until needed
- Build 1000 posts in under 5 seconds

## Tests Required

### Content Tests
- Markdown parsing (basic, extensions, edge cases)
- Frontmatter parsing (valid, missing fields, invalid YAML)
- Slug generation (from title, special chars, unicode)
- Date parsing (multiple formats, timezones)
- Draft/future filtering
- Excerpt extraction (auto, manual separator)

### Template Tests
- Variable substitution
- Template inheritance
- Include/partial loading
- Custom filters
- Loop iteration (posts, tags)
- Conditional rendering
- Missing variable handling

### Build Tests
- Full site build
- Permalink generation
- Pagination (correct posts per page, nav links)
- Taxonomy pages (tags, categories)
- Feed generation (valid RSS/Atom/JSON)
- Sitemap generation (valid XML)
- Asset copying
- CSS/JS bundling and minification
- Clean URLs

### CLI Tests
- Build command with various flags
- Init command creates structure
- New command creates files
- Serve command starts server
- Check command finds issues
- Config file loading
- Flag precedence over config

### Server Tests
- Serves built files
- 404 handling
- Clean URL resolution
- File change detection
- Live reload injection

### Integration Tests
- End-to-end: init → new → build → serve
- Build with custom theme
- Incremental rebuild
- Cross-linking between posts

## Project Structure
```
ssg/
  __init__.py
  cli.py
  config.py
  content/
    __init__.py
    markdown.py
    frontmatter.py
  build/
    __init__.py
    builder.py
    pagination.py
    taxonomy.py
    feeds.py
    sitemap.py
  templates/
    __init__.py
    engine.py
    filters.py
  assets/
    __init__.py
    css.py
    js.py
    static.py
  server/
    __init__.py
    dev_server.py
    watcher.py
    livereload.py
  utils/
    __init__.py
    slugify.py
    dates.py
themes/
  default/
    templates/
      base.html
      home.html
      post.html
      page.html
      list.html
      tag.html
      partials/
        nav.html
        footer.html
    static/
      css/
        main.css
      js/
        main.js
tests/
  test_content.py
  test_templates.py
  test_build.py
  test_cli.py
  test_server.py
  fixtures/
    sample_posts/
    sample_config.yaml
pyproject.toml
README.md
```

## Example Usage

```bash
# Initialize new site
ssg init mysite
cd mysite

# Create content
ssg new post "My First Post"
ssg new page "About Me"

# Build site
ssg build

# Development with live reload
ssg serve --drafts

# Production build
ssg build --minify --base-url https://example.com

# Validate
ssg check
```

## Deliverables
- All source code with proper package structure
- pyproject.toml with dependencies
- Default theme with responsive templates
- README.md with setup and usage guide
- Sample content (3 posts, 2 pages)
- Comprehensive test suite
