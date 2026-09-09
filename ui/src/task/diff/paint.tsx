// Painting one line of a diff: what the language says it is, and what the
// edit changed inside it.
//
// The two have to live together. Syntax highlighting alone leaves a reader
// comparing two long lines by eye, which is the work the word-level diff
// does for them; word-level alone leaves code the colour of prose. So the
// line is tokenised once by the highlighter and the tokens are then cut at
// the edges of the change, which is why this is not two functions.

import hljs from "highlight.js/lib/core";
import go from "highlight.js/lib/languages/go";
import ts from "highlight.js/lib/languages/typescript";
import js from "highlight.js/lib/languages/javascript";
import json from "highlight.js/lib/languages/json";
import bash from "highlight.js/lib/languages/bash";
import css from "highlight.js/lib/languages/css";
import xml from "highlight.js/lib/languages/xml";
import yaml from "highlight.js/lib/languages/yaml";
import md from "highlight.js/lib/languages/markdown";
import sql from "highlight.js/lib/languages/sql";
import type { Part } from "./parse";

for (const [name, language] of Object.entries({ go, ts, js, json, bash, css, xml, yaml, md, sql })) {
  hljs.registerLanguage(name, language);
}

// tongues maps what a file is called to what it is written in. A name this
// does not know is drawn as plain text, which is honest: a wrong guess
// colours a word as a keyword it is not.
const tongues: Record<string, string> = {
  go: "go", ts: "ts", tsx: "ts", js: "js", jsx: "js", mjs: "js", cjs: "js",
  json: "json", sh: "bash", bash: "bash", zsh: "bash", css: "css",
  html: "xml", xml: "xml", svg: "xml", yaml: "yaml", yml: "yaml",
  md: "md", sql: "sql",
};

export function tongue(name: string): string | undefined {
  return tongues[name.split(".").pop()?.toLowerCase() ?? ""];
}

// A token is a run of the line the highlighter gave one class.
interface Token {
  text: string;
  cls: string;
}

// seen is every line already worked out, by its text and its language.
//
// Highlighting is the expensive thing in a diff — a grammar run and a DOM
// parse per line — and React repaints a file whenever anything about it
// changes: a wrap toggle, a line picked, a hunk opened. Without this, a
// six-thousand-line file cost most of a second on every one of those, and a
// reader clicking around several of them took the tab down with them.
const seen = new Map<string, Token[]>();

// A bound, because this outlives every diff the tab has shown. It is cleared
// rather than trimmed: a cache with an eviction order is a second thing to
// get right, and losing it costs one repaint.
const most = 40000;

// One holder for every line, rather than one per line: the parse is what
// costs, but a discarded element per rendered row is garbage this loop does
// not need to make.
let holder: HTMLDivElement | undefined;

// tokens is the line, cut into what the language calls each part of it.
//
// The highlighter answers HTML, so it is read back through the DOM rather
// than with a regular expression: nested spans are ordinary in a grammar,
// and a pattern that got them wrong would silently mis-colour code.
function tokens(text: string, language?: string): Token[] {
  if (!language || !text) return [{ text, cls: "" }];

  const key = `${language}\u0000${text}`;
  const had = seen.get(key);
  if (had) return had;

  let html: string;
  try {
    html = hljs.highlight(text, { language, ignoreIllegals: true }).value;
  } catch {
    return [{ text, cls: "" }];
  }

  holder ??= document.createElement("div");
  holder.innerHTML = html;

  const out: Token[] = [];

  const walk = (node: Node, cls: string) => {
    for (const child of Array.from(node.childNodes)) {
      if (child.nodeType === Node.TEXT_NODE) {
        out.push({ text: child.textContent ?? "", cls });
      } else if (child instanceof HTMLElement) {
        walk(child, child.className || cls);
      }
    }
  };

  walk(holder, "");

  if (seen.size > most) seen.clear();
  seen.set(key, out);

  return out;
}

// changed is where the edit is, as offsets into the line: the parser gives
// the change as three pieces, and the middle one is what moved.
function changed(parts?: Part[]): [number, number] | undefined {
  if (!parts) return undefined;

  let at = 0;

  for (const part of parts) {
    if (part.changed) return [at, at + part.text.length];
    at += part.text.length;
  }

  return undefined;
}

// paint is the line, ready to draw: coloured by the language, with the part
// the edit touched standing out from the rest of it.
export function paint(
  text: string,
  parts: Part[] | undefined,
  language: string | undefined,
  strong: string,
): React.ReactNode[] {
  const range = changed(parts);
  const out: React.ReactNode[] = [];

  let at = 0;
  let key = 0;

  for (const token of tokens(text, language)) {
    // A token can straddle either edge of the change, so it is cut there and
    // every piece keeps the colour the grammar gave it.
    //
    // The cuts are gathered before any of them is applied: taken one at a
    // time against a list that is being spliced, the second cut is measured
    // against the first one's boundary and never lands.
    const start = at;
    const end = at + token.text.length;
    const edges = [start, end];

    if (range) {
      for (const cut of range) {
        if (cut > start && cut < end) edges.push(cut);
      }

      edges.sort((a, b) => a - b);
    }

    for (let i = 0; i < edges.length - 1; i++) {
      const from = edges[i] ?? 0;
      const to = edges[i + 1] ?? 0;
      const piece = token.text.slice(from - at, to - at);
      if (!piece) continue;

      const inside = range && from >= range[0] && to <= range[1];

      out.push(
        <span key={key++} className={`${token.cls} ${inside ? strong : ""}`.trim() || undefined}>
          {piece}
        </span>,
      );
    }

    at += token.text.length;
  }

  return out.length > 0 ? out : [" "];
}
