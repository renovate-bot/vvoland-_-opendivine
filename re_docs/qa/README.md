# Questions and answers

This directory contains focused questions and answers about the
reverse-engineered game. Each file is one self-contained Q&A entry.

## File format

Use a descriptive, lowercase, hyphen-separated filename. Put the question
first, followed by a Markdown horizontal rule, then the answer:

```markdown
The question goes here?

---

The answer goes here.
```

The answer may contain headings, lists, code blocks, and links to the detailed
reverse-engineering notes. Keep the question focused and put supporting
technical detail in the answer.

## Entries

- [game-loop.md](game-loop.md) — how the main game loop updates, renders, and
  handles performance drops.
