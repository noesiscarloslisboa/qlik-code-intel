# Statement boundaries and backtick identifiers

The user approved the next hardening pass: recover labeled LOADs after unfinished
unsupported statements, support backtick identifiers, then validate retrieval.
Known-valid application scripts are not available yet; the user may provide them
later. Complete independent-fixture and pinned-reference checks now, and explicitly
leave application-level acceptance open.

## Recovery

Keep unknown statement bodies opaque, including dollar expansions and quoted text.
Replace the greedy fallback body with a stateless Tree-sitter external scanner.
It terminates at an unquoted semicolon or recovers before a line-start table label
outside quotes, comments, bracketed content, and dollar expansions. Indentation,
CRLF, Unicode identifiers, quoted names, and symbolic labels are allowed. A colon
followed by a slash is a URI component, not a recovery boundary. Do not recover at
unlabeled or same-line LOAD text, which may belong to the unsupported statement.

The scanner emits distinct terminated/unterminated body tokens. Preserve the
`unsupported` diagnostic and add `unterminated-statement` when a boundary/EOF is
reached without a semicolon. Recovered LOADs retain exact source positions. Their
syntax is indexed without claiming that malformed surrounding code would execute.
Unsupported nodes break preceding LOAD adjacency and contribute no query symbols.
Use iterative protected-context tracking so deeply nested macro text cannot exhaust
the C stack. Disable external scanning in generic Tree-sitter error recovery.

This is parser logic, not source preprocessing or regex-based symbol extraction.
A pure greedy regex cannot protect nested contexts and distinguish line-start
labels; reparsing source slices would complicate locations and query behavior.
The scanner adds no dependency and is compiled with the included generated parser.

## Quoting

Add `backtick_name` beside bracket/double-quoted names. Support dollar expansions,
labels, field inputs/aliases, variables, table references, data/include paths,
SET text, and format options. Preserve raw query capture text; CLI normalization
removes enclosing backticks. Literal strings and raw SET values retain their roles.

The initial doubled-backtick assumption was corrected during documentation review:
[Qlik's quotation rules](https://help.qlik.com/en-US/cloud-services/Subsystems/Hub/Content/Sense_Hub/Scripting/use-quotes-in-script.htm)
list double quotes, brackets, and single quotes as escape delimiters, excluding
backticks. Do not add that undocumented escape. Test doubled double quotes and
alternative quotation around literal backticks instead.

## Verification

Add failing original Go regressions before implementation; inspect new corpus trees
before updating expectations. Check opaque strings/comments/macros for phantom
symbols, EOF behavior, nested input, label/source positions, query captures, and
the distinction between labeled and unlabeled recovery. Test backtick names and documented quote escaping,
symbols, edges, and maps/context budgets in a representative multi-file fixture.
Run parser generation/reproducibility, grammar corpus, fresh race/executable tests,
formatting, vet, and build. Re-run the pinned external audit, retaining diagnostics
and reporting changed counts candidly. No GPL samples are incorporated.
