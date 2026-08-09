// Unit tests for Readflow extension utility functions.
// Run: node extension/test.js

var passed = 0;
var failed = 0;

function assert(cond, msg) {
  if (cond) {
    passed++;
  } else {
    failed++;
    console.error('FAIL: ' + msg);
  }
}

function assertEqual(actual, expected, msg) {
  if (actual === expected) {
    passed++;
  } else {
    failed++;
    console.error('FAIL: ' + msg + ' — expected: ' + JSON.stringify(expected) + ', got: ' + JSON.stringify(actual));
  }
}

// ========== Copied from background.js ==========

function sanitizeFilename(name) {
  return name
    .replace(/[<>:"/\\|?*]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 200) || 'untitled';
}

function slug(s) {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
}

function encodePath(p) {
  return p.split('/').map(function (s) { return encodeURIComponent(s); }).join('/');
}

function buildUri(vault, filePath, content) {
  var params = [];
  if (vault) {
    params.push('vault=' + encodeURIComponent(vault));
  }
  params.push('file=' + encodePath(filePath));
  params.push('content=' + encodeURIComponent(content));
  return 'obsidian://new?' + params.join('&');
}

function pushField(lines, name, value) {
  var escaped = String(value)
    .replace(/\\/g, '\\\\')
    .replace(/"/g, '\\"')
    .replace(/\n/g, '\\n')
    .replace(/\r/g, '');
  lines.push(name + ': "' + escaped + '"');
}

function formatArticleMarkdown(article, lite) {
  lite = lite || false;
  var lines = [];
  lines.push('---');
  pushField(lines, 'title', article.title || 'Untitled');
  if (article.author) pushField(lines, 'author', article.author);
  if (article.url) pushField(lines, 'source', article.url);
  if (article.site_name) pushField(lines, 'site', article.site_name);

  var tags = ['readflow'];
  if (article.site_name) tags.push(slug(article.site_name));
  lines.push('tags: [' + tags.join(', ') + ']');

  if (article.created_at) {
    pushField(lines, 'date', article.created_at.slice(0, 10));
  }
  lines.push('---');
  lines.push('');
  lines.push('# ' + (article.title || 'Untitled'));
  lines.push('');

  if (lite) {
    lines.push('<!-- Paste full content here (Cmd+V / Ctrl+V) -->');
  } else if (article.content_markdown) {
    lines.push(article.content_markdown);
  }

  return lines.join('\n');
}

// ========== Tests ==========

console.log('=== sanitizeFilename ===');

assertEqual(sanitizeFilename('Hello World'), 'Hello World', 'preserves spaces');
assertEqual(sanitizeFilename('Hello:World'), 'HelloWorld', 'removes colon');
assertEqual(sanitizeFilename('Hello/World'), 'HelloWorld', 'removes slash');
assertEqual(sanitizeFilename('Hello\\World'), 'HelloWorld', 'removes backslash');
assertEqual(sanitizeFilename('<Hello>'), 'Hello', 'removes angle brackets');
assertEqual(sanitizeFilename('Hello|World'), 'HelloWorld', 'removes pipe');
assertEqual(sanitizeFilename('Hello?World'), 'HelloWorld', 'removes question mark');
assertEqual(sanitizeFilename('Hello*World'), 'HelloWorld', 'removes asterisk');
assertEqual(sanitizeFilename('  Hello   World  '), 'Hello World', 'collapses whitespace');
assertEqual(sanitizeFilename(''), 'untitled', 'empty returns untitled');
assertEqual(sanitizeFilename('<>:"/\\|?*'), 'untitled', 'all special chars returns untitled');

var longName = 'a'.repeat(250);
assert(sanitizeFilename(longName).length <= 200, 'long names truncated to 200 chars');

console.log('=== slug ===');

assertEqual(slug('Example Site'), 'example-site', 'converts to lowercase slug');
assertEqual(slug('Hacker News'), 'hacker-news', 'handles multiple words');
assertEqual(slug('  Spaces  '), 'spaces', 'trims spaces');
assertEqual(slug('Special!@#Chars'), 'special-chars', 'removes special chars');
assertEqual(slug('Already-Slug'), 'already-slug', 'keeps hyphens');

console.log('=== encodePath ===');

assertEqual(encodePath('Readflow/Note.md'), 'Readflow/Note.md', 'preserves slash');
assertEqual(encodePath('My Folder/My Note.md'), 'My%20Folder/My%20Note.md', 'encodes spaces in segments');
assertEqual(encodePath('a/b/c'), 'a/b/c', 'handles nested paths');

console.log('=== buildUri ===');

var uri1 = buildUri('MyVault', 'Readflow/Note.md', '# Hello');
assert(uri1.startsWith('obsidian://new?'), 'starts with obsidian://new scheme');
assert(uri1.includes('vault=MyVault'), 'includes vault param');
assert(uri1.includes('file=Readflow/Note.md'), 'file path has literal slashes');
assert(uri1.includes('content=' + encodeURIComponent('# Hello')), 'content is encoded');

var uri2 = buildUri('', 'Readflow/Note.md', 'hi');
assert(!uri2.includes('vault='), 'omits vault when empty');
assert(uri2.includes('file=Readflow/Note.md'), 'includes file without vault');

var uri3 = buildUri('My Vault', 'Folder/Title with spaces.md', 'content');
assert(uri3.includes('vault=My%20Vault'), 'encodes vault with spaces');
assert(uri3.includes('Folder/Title%20with%20spaces.md'), 'encodes spaces in filename');

console.log('=== formatArticleMarkdown (normal) ===');

var article = {
  title: 'Test Article',
  author: 'Jane Doe',
  url: 'https://example.com/test',
  site_name: 'Example Blog',
  created_at: '2024-03-15T10:30:00Z',
  content_markdown: 'This is the article body.\n\nSecond paragraph.'
};

var md = formatArticleMarkdown(article);

assert(md.startsWith('---'), 'starts with frontmatter delimiter');
assert(md.includes('title: "Test Article"'), 'includes title field');
assert(md.includes('author: "Jane Doe"'), 'includes author field');
assert(md.includes('source: "https://example.com/test"'), 'includes source field');
assert(md.includes('site: "Example Blog"'), 'includes site field');
assert(md.includes('tags: [readflow, example-blog]'), 'includes auto-generated tags');
assert(md.includes('date: "2024-03-15"'), 'includes date field');
assert(md.includes('# Test Article'), 'includes heading');
assert(md.includes('This is the article body.'), 'includes body content');
assert(md.includes('Second paragraph.'), 'includes second paragraph');

console.log('=== formatArticleMarkdown (lite) ===');

var lite = formatArticleMarkdown(article, true);
assertEqual(lite.includes('This is the article body.'), false, 'lite mode excludes body');
assert(lite.includes('<!-- Paste full content here'), 'lite mode includes paste placeholder');
assert(lite.includes('title: "Test Article"'), 'lite mode still has metadata');

console.log('=== formatArticleMarkdown (edge cases) ===');

var minArticle = { title: 'Minimal' };
var minMd = formatArticleMarkdown(minArticle);
assert(minMd.includes('title: "Minimal"'), 'minimal article');
assert(minMd.includes('# Minimal'), 'minimal heading');
assert(!minMd.includes('author:'), 'no author field when missing');

var specialTitle = { title: 'Say "Hello" \\ World' };
var specialMd = formatArticleMarkdown(specialTitle);
assert(specialMd.includes('title: "Say \\"Hello\\" \\\\ World"'), 'escapes quotes and backslashes in title');

var newlineTitle = { title: 'Line 1\nLine 2' };
var nlMd = formatArticleMarkdown(newlineTitle);
assert(nlMd.includes('Line 1\\nLine 2'), 'escapes newlines in title');

// Overlong tag site name
var weirdSite = { title: 'X', site_name: 'A Very Long Site Name!!! With Special Chars' };
var weirdMd = formatArticleMarkdown(weirdSite);
assert(weirdMd.includes('a-very-long-site-name-with-special-chars'), 'slugs long site names correctly');

// ========== URI length threshold test ==========
console.log('=== URI length threshold ===');

// Small content: URI should be under 1.5MB
var smallUri = buildUri('Vault', 'Folder/Note.md', 'Short content');
assert(smallUri.length < 1500000, 'small content fits within URI limit');

// Large content: ~2MB should exceed threshold
var largeContent = 'x'.repeat(2000000);
var largeUri = buildUri('Vault', 'Folder/Note.md', largeContent);
assert(largeUri.length > 1500000, 'large content exceeds URI threshold');

// ========== End-to-end simulation: clipToObsidian pipeline ==========
console.log('=== E2E: clipToObsidian pipeline ===');

function simulateClip(article, vault, folder) {
  var datePrefix = article.created_at ? article.created_at.slice(0, 10) : '';
  var filename = (datePrefix ? datePrefix + ' ' : '') + sanitizeFilename(article.title) + '.md';
  var filePath = folder.replace(/\/$/, '') + '/' + filename;
  var markdown = formatArticleMarkdown(article);
  var uri = buildUri(vault, filePath, markdown);

  if (uri.length > 1500000) {
    var liteMarkdown = formatArticleMarkdown(article, true);
    var liteUri = buildUri(vault, filePath, liteMarkdown);
    return { success: true, useClipboard: true, markdown: markdown, obsidianUri: liteUri };
  }
  return { success: true, obsidianUri: uri };
}

// E2E: Normal article with folder param
var normalArticle = {
  title: 'Normal Article',
  author: 'Author',
  url: 'https://example.com/normal',
  site_name: 'Example',
  created_at: '2024-06-01T00:00:00Z',
  content_markdown: 'Hello world.\n\nThis is a paragraph.'
};
var normalResp = simulateClip(normalArticle, 'MyVault', 'Readflow');

assertEqual(normalResp.success, true, 'E2E normal: success is true');
assertEqual(normalResp.useClipboard, undefined, 'E2E normal: no clipboard fallback');
assert(normalResp.obsidianUri.startsWith('obsidian://new?'), 'E2E normal: returns URI');
assert(normalResp.obsidianUri.includes('vault=MyVault'), 'E2E normal: vault in URI');
assert(normalResp.obsidianUri.includes('file=Readflow/'), 'E2E normal: folder in file path');
assert(normalResp.obsidianUri.includes('2024-06-01%20Normal%20Article.md'), 'E2E normal: date-prefixed filename');

// E2E: No vault (uses active vault)
var noVaultResp = simulateClip(normalArticle, '', 'Readflow');
assert(!noVaultResp.obsidianUri.includes('vault='), 'E2E no vault: omits vault param');

// E2E: Multiple folders — Work vs Personal
var workResp = simulateClip(normalArticle, 'V', 'Work');
var personalResp = simulateClip(normalArticle, 'V', 'Personal');

assert(workResp.obsidianUri.includes('file=Work/'), 'E2E Work folder in URI');
assert(personalResp.obsidianUri.includes('file=Personal/'), 'E2E Personal folder in URI');
assert(workResp.obsidianUri !== personalResp.obsidianUri, 'E2E different folders produce different URIs');

// E2E: Nested custom folder
var nestedResp = simulateClip(normalArticle, 'V', 'MyArticles/2024');
assert(nestedResp.obsidianUri.includes('file=MyArticles/2024/'), 'E2E nested folder path in URI');

// E2E: Huge article → clipboard fallback
var hugeArticle = {
  title: 'Huge Article',
  created_at: '2024-01-01T00:00:00Z',
  content_markdown: 'x'.repeat(1900000)
};
var hugeResp = simulateClip(hugeArticle, 'MyVault', 'Readflow');

assertEqual(hugeResp.success, true, 'E2E huge: success is true');
assertEqual(hugeResp.useClipboard, true, 'E2E huge: triggers clipboard fallback');
assertEqual(typeof hugeResp.markdown, 'string', 'E2E huge: markdown returned for clipboard');
assert(hugeResp.markdown.includes('Huge Article'), 'E2E huge: full markdown has heading');
assert(hugeResp.obsidianUri.length < 1500000, 'E2E huge: lite URI under threshold');
assert(!hugeResp.obsidianUri.includes('xxxxx'), 'E2E huge: lite URI has no body');
assert(hugeResp.obsidianUri.includes('Paste%20full%20content'), 'E2E huge: lite URI has placeholder');

// E2E: Article without created_at → no date prefix
var noDateArticle = {
  title: 'No Date',
  content_markdown: 'Body.'
};
var noDateResp = simulateClip(noDateArticle, 'V', 'Readflow');
assert(noDateResp.obsidianUri.includes('file=Readflow/No%20Date.md'), 'E2E no date: no date prefix');

// ========== Results ==========
console.log('');
console.log('Passed: ' + passed);
console.log('Failed: ' + failed);

if (failed > 0) {
  process.exit(1);
}
