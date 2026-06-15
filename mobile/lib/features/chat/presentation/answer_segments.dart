/// Splits an assistant answer (Markdown) into renderable segments so the chat
/// can lift the actionable "Что делать" section out of the prose and show it as
/// a highlighted step card (see [StepsSegment] / `ActionStepsCard`).
///
/// The agronomist prompt (`system_v1.md`) always answers in three sections, the
/// last being **Что делать** — numbered, concrete steps. We detect that heading
/// and the list that follows it; everything else stays plain Markdown. Parsing
/// runs on every rebuild (including mid-stream), so a half-arrived list still
/// renders as a card and fills in as deltas land.
///
/// The heading is matched as either an ATX heading (`### Что делать`) or the
/// bold label the prompt mandates (`**Что делать**`, optionally with a trailing
/// emoji, separator or an inline list). Nested sub-items fold into their parent
/// step; a stray note or non-indented paragraph ends the list rather than being
/// swallowed into a badge.
library;

/// One renderable piece of an assistant answer.
sealed class AnswerSegment {
  const AnswerSegment();
}

/// A run of plain Markdown rendered as-is.
class MarkdownSegment extends AnswerSegment {
  const MarkdownSegment(this.text);

  final String text;
}

/// The actionable steps section, rendered as a highlighted card. [title] is the
/// heading the model used (e.g. "Что делать"); [steps] are the list items in
/// order, each still carrying inline Markdown (bold, nested sub-lists, etc.).
class StepsSegment extends AnswerSegment {
  const StepsSegment({required this.title, required this.steps});

  final String title;
  final List<String> steps;
}

/// Heading phrases that introduce the actionable section. Matched case-folded
/// against the cleaned heading text; a heading counts when it equals or starts
/// with one of these (so "Что делать сейчас" / "Что сделать на этой неделе"
/// both qualify). Deliberately narrow to avoid cardifying "Возможные причины".
const _actionTriggers = <String>[
  'что делать',
  'что сделать',
  'что нужно делать',
  'что нужно сделать',
  'что предпринять',
  'план действий',
];

final _atxHeading = RegExp(r'^#{1,6}\s+(.*\S.*)$');
final _boldSpan = RegExp(r'^\*\*(.+?)\*\*(.*)$');
final _listItem = RegExp(r'^\s*(?:\d{1,2}[.)]|[-*•·])\s+(\S.*)$');
final _inlineListStart = RegExp(r'^\d{1,2}[.)]\s');
final _inlineListMarker = RegExp(r'(?:^|\s)\d{1,2}[.)]\s+');

/// A detected action heading: its display [title] and any inline content left on
/// the heading line after the label (used to pick up an inline list).
typedef _Heading = ({String title, String trailing});

/// Parses [text] into ordered segments. Without a detectable action section the
/// whole input comes back as a single [MarkdownSegment], so callers can render
/// the result the same way regardless.
List<AnswerSegment> parseAnswerSegments(String text) {
  final lines = text.split('\n');
  final segments = <AnswerSegment>[];
  final pre = <String>[];

  void flushPre() {
    if (pre.isEmpty) {
      return;
    }
    final body = pre.join('\n').trim();
    if (body.isNotEmpty) {
      segments.add(MarkdownSegment(body));
    }
    pre.clear();
  }

  var i = 0;
  while (i < lines.length) {
    final heading = _actionHeading(lines[i]);
    if (heading != null) {
      // Steps written inline on the heading line, e.g. "**Что делать:** 1. …".
      final inline = _inlineSteps(heading.trailing);
      if (inline.isNotEmpty) {
        flushPre();
        segments.add(StepsSegment(title: heading.title, steps: inline));
        i++;
        continue;
      }
      // Otherwise the list must follow on its own lines (blanks allowed).
      var j = i + 1;
      while (j < lines.length && lines[j].trim().isEmpty) {
        j++;
      }
      if (j < lines.length && _listItem.hasMatch(lines[j])) {
        final (steps, end) = _collectSteps(lines, j);
        if (steps.isNotEmpty) {
          flushPre();
          segments.add(StepsSegment(title: heading.title, steps: steps));
          i = end;
          continue;
        }
      }
    }
    pre.add(lines[i]);
    i++;
  }

  flushPre();
  return segments;
}

/// Collects the list starting at [start], returning the step texts and the index
/// of the first line past the list. Items indented past the first item fold into
/// the current step as a nested sub-list; an indented prose line is treated as a
/// wrapped continuation; a stray note/heading or a non-indented paragraph ends
/// the list so it is never swallowed into a badge.
(List<String>, int) _collectSteps(List<String> lines, int start) {
  final steps = <String>[];
  final firstIndent = _indent(lines[start]);
  String? current;
  var k = start;

  while (k < lines.length) {
    final line = lines[k];
    final item = _listItem.firstMatch(line);
    if (item != null) {
      if (current != null && _indent(line) >= firstIndent + 2) {
        current = '$current\n${line.trim()}'; // nested sub-item of the step
        k++;
        continue;
      }
      if (current != null) {
        steps.add(current.trim());
      }
      current = item.group(1)!.trim();
      k++;
      continue;
    }

    final trimmed = line.trim();
    if (trimmed.isEmpty) {
      var m = k + 1;
      while (m < lines.length && lines[m].trim().isEmpty) {
        m++;
      }
      if (m < lines.length && _listItem.hasMatch(lines[m])) {
        k = m; // inter-item blank line — keep going
        continue;
      }
      break; // blank line ends the list
    }

    if (current == null) {
      break;
    }
    if (trimmed.startsWith('**') || _atxHeading.hasMatch(trimmed)) {
      break; // a note ("**Важно:** …") or next heading ends the list
    }
    if (_indent(line) >= firstIndent + 2) {
      current = '$current $trimmed'; // indented wrapped continuation
      k++;
      continue;
    }
    break; // a non-indented paragraph ends the list
  }

  if (current != null) {
    steps.add(current.trim());
  }
  return (steps, k);
}

/// Returns the heading when [line] introduces the actionable section, else null.
/// Handles ATX (`### Что делать`) and the bold label form the prompt uses
/// (`**Что делать**`), tolerating a trailing emoji/separator/inline content.
_Heading? _actionHeading(String line) {
  final trimmed = line.trim();
  String? inner;
  var trailing = '';

  final atx = _atxHeading.firstMatch(trimmed);
  if (atx != null) {
    inner = atx.group(1);
  } else if (trimmed.startsWith('**')) {
    final bold = _boldSpan.firstMatch(trimmed);
    if (bold != null) {
      inner = bold.group(1);
      trailing = bold.group(2) ?? '';
    }
  }

  if (inner == null) {
    return null;
  }
  final title = _cleanHeading(inner);
  return _isActionTitle(title) ? (title: title, trailing: trailing) : null;
}

/// Splits an inline list left on a heading line ("1. … 2. … 3. …") into steps.
/// Returns an empty list when [trailing] does not begin with a list marker, so
/// a plain "— описание" or a stray emoji never produces a card.
List<String> _inlineSteps(String trailing) {
  final t = trailing.trim();
  if (!_inlineListStart.hasMatch(t)) {
    return const [];
  }
  final markers = _inlineListMarker.allMatches(t).toList();
  final steps = <String>[];
  for (var i = 0; i < markers.length; i++) {
    final from = markers[i].end;
    final to = i + 1 < markers.length ? markers[i + 1].start : t.length;
    final step = t.substring(from, to).trim();
    if (step.isNotEmpty) {
      steps.add(step);
    }
  }
  return steps;
}

/// Strips emphasis/heading markers and trailing separators from a heading's
/// inner text, leaving the human-readable title (a trailing emoji survives).
String _cleanHeading(String raw) {
  return raw
      .trim()
      .replaceAll(RegExp(r'^[*_#\s]+'), '')
      .replaceAll(RegExp(r'[*_\s:：—–-]+$'), '')
      .trim();
}

bool _isActionTitle(String title) {
  final low = title.toLowerCase();
  return _actionTriggers.any((t) => low == t || low.startsWith('$t '));
}

int _indent(String line) => line.length - line.trimLeft().length;
