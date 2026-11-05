# Problem record test vectors

`v1.txt` holds the shared test vectors of schema `frostyard.problem/1`. The
Go package `github.com/frostyard/std/problems` and the Rust crate
`frostyard-problems` (`rust/`) both read every vector, so the 2 writers and
the 2 readers agree. The contract is
[docs/specs/problems-package.md](../../../docs/specs/problems-package.md),
which follows frostyard/core `docs/specs/problem-report.md`.

## Format

The format is plain text, so that a test reads it with no JSON or serde
library.

- The file is UTF-8 with `\n` line ends.
- 1 blank line separates 2 blocks. 1 block is 1 vector.
- A line that starts with `#` is a comment. A block with only comments
  (the file header) is not a vector.
- Each other line is `key: value`. Split it at the first `: `. The value is
  the rest of the line, as is (no trim). A line `key:` with nothing after the
  colon has an empty value.
- A key can repeat. The order of the lines is significant.

## Keys

| Key | Vector types | Meaning |
| --- | --- | --- |
| `vector` | all | The name of the vector, unique in the file |
| `type` | all | `write`, `validate` or `frame` |
| `in.app_id`, `in.version`, `in.component` | `write` | The identity given to the writer |
| `in.runtime` | `write` | `go` or `rust`: only the writer of that runtime writes the vector; every reader validates its journal fields |
| `in.class` | `write` | `error` or `panic` |
| `in.code` | `write` | The code of an error |
| `in.message` | `write` | The error text or the panic message. No `in.message` on an error: the writer gets no error value (Go: `nil`), and `MESSAGE` is the code |
| `in.frame` | `write`, `frame` | A raw Go function name, as the runtime gives it, innermost first; it repeats. The writer maps each one |
| `in.location` | `write` | The Rust panic location |
| `out.frame` | `frame` | The result of the frame mapping (`MapFrame` in Go) for `in.frame` |
| `journal.<FIELD>` | `write`, `validate` | 1 journal field. In a `write` vector, the lines list every field in the order the writer writes them. Repeated `journal.FROSTYARD_FRAMES` lines join with 1 `\n` into 1 field value |
| `payload.<field>` | `write`, `validate` | 1 S8 detail field that the reader returns when it accepts the record. `payload.frames` repeats, 1 line for each frame |
| `expect` | `write`, `validate` | `ok` or `reject`. A `write` vector with `reject` writes nothing (a bad code, or an inert writer). A `validate` vector with `reject` is a record that every reader drops |

## How a test uses a vector

- `write`, runtime of the test: give the `in.*` values to the writer. With
  `expect: ok`, the written fields must equal the `journal.*` lines, in
  order, with no other field. With `expect: reject`, the writer writes
  nothing.
- `write` (both runtimes) and `validate` with `expect: ok`: the reader
  validates the `journal.*` fields and returns exactly the `payload.*`
  fields; no other detail field.
- `validate` with `expect: reject`: the reader rejects the `journal.*`
  fields.
- `frame`: the mapping of `in.frame` gives `out.frame`.
