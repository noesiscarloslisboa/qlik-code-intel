// Independently authored recovery scanner; see grammar.js and README.md.
#include "tree_sitter/parser.h"
#include "tree_sitter/array.h"

enum QlikToken {
  UNSUPPORTED_BODY,
  UNTERMINATED_UNSUPPORTED_BODY,
  NAME_FRAGMENT,
  IMMEDIATE_NAME_FRAGMENT,
  ERROR_SENTINEL,
};

typedef struct {
  int32_t close;
  uint32_t depth;
} QlikGroup;

static void qlik_advance(TSLexer *lexer) { lexer->advance(lexer, false); }

// Called after consuming the first slash. Leave a line comment's newline for
// the caller, which owns line-start detection. Block comments protect labels.
static void qlik_comment(TSLexer *lexer) {
  if (lexer->lookahead == '/') {
    while (!lexer->eof(lexer) && lexer->lookahead != '\n') qlik_advance(lexer);
  } else if (lexer->lookahead == '*') {
    qlik_advance(lexer);
    while (!lexer->eof(lexer)) {
      int32_t c = lexer->lookahead;
      qlik_advance(lexer);
      if (c == '*' && lexer->lookahead == '/') {
        qlik_advance(lexer);
        break;
      }
    }
  }
}

// Protect quotes, brackets, and dollar expansions, including quotes nested
// inside an expansion in a quoted value. Use a stack rather than C recursion.
// For expansions, lookahead is '(' and close is ')'.
static void qlik_group(TSLexer *lexer, int32_t close) {
  Array(QlikGroup) stack = array_new();
  array_push(&stack, ((QlikGroup){close, 1}));
  qlik_advance(lexer);
  while (stack.size && !lexer->eof(lexer)) {
    QlikGroup *top = array_back(&stack);
    int32_t c = lexer->lookahead;
    if (c == '$') {
      qlik_advance(lexer);
      if (lexer->lookahead == '(') {
        array_push(&stack, ((QlikGroup){')', 1}));
        qlik_advance(lexer);
      }
    } else if (top->close == ')') {
      qlik_advance(lexer);
      if (c == '(') {
        top->depth++;
      } else if (c == ')') {
        if (--top->depth == 0) array_pop(&stack);
      } else if (c == '\'' || c == '"' || c == '`' || c == '[') {
        array_push(&stack, ((QlikGroup){c == '[' ? ']' : c, 1}));
      } else if (c == '/') {
        qlik_comment(lexer);
      }
    } else {
      qlik_advance(lexer);
      if (c == top->close) {
        if (lexer->lookahead == c) qlik_advance(lexer); // Doubled delimiter.
        else array_pop(&stack);
      }
    }
  }
  array_delete(&stack);
}

static bool qlik_name_start(int32_t c) {
  return c == '%' || c == '@' || c == '_' || (c >= 'a' && c <= 'z') ||
         (c >= 'A' && c <= 'Z') || (c >= 0x00c0 && c <= 0xffff);
}

static bool qlik_name_continue(int32_t c) {
  return qlik_name_start(c) || (c >= '0' && c <= '9') || c == '.' || c == '$';
}

static bool qlik_space(int32_t c) {
  // Match grammar extras (Tree-sitter's \\s is ASCII whitespace).
  return c == ' ' || (c >= '\t' && c <= '\r') || c == 0xfeff;
}

// Consume a name and identify REM only as a whole comment keyword.
static bool qlik_read_name(TSLexer *lexer) {
  unsigned length = 0;
  bool rem = true;
  const char expected[] = "REM";
  do {
    int32_t c = lexer->lookahead;
    if (length >= 3 || (c != expected[length] && c != expected[length] + ('a' - 'A'))) rem = false;
    length++;
    qlik_advance(lexer);
    if (c == '$' && lexer->lookahead == '(') {
      qlik_group(lexer, ')');
      rem = false;
    }
  } while (qlik_name_continue(lexer->lookahead));
  int32_t c = lexer->lookahead;
  return rem && length == 3 && (c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == ';');
}

static void qlik_rem_comment(TSLexer *lexer) {
  while (!lexer->eof(lexer) && lexer->lookahead != ';') qlik_advance(lexer);
  if (!lexer->eof(lexer)) qlik_advance(lexer);
}

static bool qlik_label_suffix(TSLexer *lexer) {
  while (qlik_space(lexer->lookahead) && lexer->lookahead != '\n') qlik_advance(lexer);
  if (lexer->lookahead != ':') return false;
  qlik_advance(lexer);
  if (lexer->lookahead == '/') {
    qlik_advance(lexer); // A URI scheme is not a table label or a // comment.
    return false;
  }
  return true;
}

// Speculate from a line start. A successful probe rewinds to mark_end when
// returned as an external token; failed probes are already opaque body text.
static bool qlik_label(TSLexer *lexer) {
  int32_t c = lexer->lookahead;
  if (qlik_name_start(c)) {
    if (qlik_read_name(lexer)) {
      qlik_rem_comment(lexer);
      return false;
    }
  } else if (c == '[' || c == '"' || c == '`') {
    qlik_group(lexer, c == '[' ? ']' : c);
  } else if (c == '$') {
    qlik_advance(lexer);
    if (lexer->lookahead != '(') return false;
    qlik_group(lexer, ')');
    if (qlik_name_continue(lexer->lookahead)) qlik_read_name(lexer);
  } else {
    return false;
  }
  return qlik_label_suffix(lexer);
}

// Emit only the literal prefix immediately before an expansion. This external
// token outranks the greedy identifier's trailing '$', while ordinary dollar
// names fall back to the unchanged internal identifier token.
static bool qlik_name_fragment(TSLexer *lexer, bool immediate) {
  if (!immediate) {
    while (qlik_space(lexer->lookahead)) lexer->advance(lexer, true);
  }
  if (immediate ? !qlik_name_continue(lexer->lookahead) : !qlik_name_start(lexer->lookahead)) return false;
  bool has_fragment = false;
  while (qlik_name_continue(lexer->lookahead)) {
    int32_t c = lexer->lookahead;
    if (c == '$') lexer->mark_end(lexer);
    qlik_advance(lexer);
    if (c == '$' && lexer->lookahead == '(') {
      if (!has_fragment) return false; // Adjacent expansion: use its internal token.
      lexer->result_symbol = immediate ? IMMEDIATE_NAME_FRAGMENT : NAME_FRAGMENT;
      return true;
    }
    has_fragment = true;
  }
  return false;
}

void *tree_sitter_qlik_external_scanner_create(void) { return NULL; }
void tree_sitter_qlik_external_scanner_destroy(void *payload) { (void)payload; }
unsigned tree_sitter_qlik_external_scanner_serialize(void *payload, char *buffer) {
  (void)payload; (void)buffer; return 0;
}
void tree_sitter_qlik_external_scanner_deserialize(void *payload, const char *buffer, unsigned length) {
  (void)payload; (void)buffer; (void)length;
}

bool tree_sitter_qlik_external_scanner_scan(void *payload, TSLexer *lexer, const bool *valid_symbols) {
  (void)payload;
  // Opt out of Tree-sitter's generic error-recovery state (all tokens valid).
  if (valid_symbols[ERROR_SENTINEL]) return false;
  if (valid_symbols[NAME_FRAGMENT]) return qlik_name_fragment(lexer, false);
  if (valid_symbols[IMMEDIATE_NAME_FRAGMENT]) return qlik_name_fragment(lexer, true);
  if (!valid_symbols[UNSUPPORTED_BODY] && !valid_symbols[UNTERMINATED_UNSUPPORTED_BODY]) return false;

  bool line_start = false;
  bool body_started = false;
  int32_t previous = 0;
  // A bare root identifier may instead be a supported table label. External
  // tokens outrank internal punctuation, so leave its colon to the grammar.
  for (;;) {
    if (qlik_space(lexer->lookahead)) {
      if (lexer->lookahead == '\n') {
        qlik_advance(lexer);
        lexer->mark_end(lexer);
        line_start = true;
      } else qlik_advance(lexer);
    } else if (lexer->lookahead == '/') {
      qlik_advance(lexer);
      if (lexer->lookahead != '/' && lexer->lookahead != '*') {
        body_started = true;
        break;
      }
      qlik_comment(lexer);
    } else if (lexer->lookahead == 'R' || lexer->lookahead == 'r') {
      // REM comments are grammar extras too. A failed probe may itself be the
      // next line-start label (including short names R and RE).
      if (!qlik_read_name(lexer)) {
        if (line_start && qlik_label_suffix(lexer)) {
          if (!valid_symbols[UNTERMINATED_UNSUPPORTED_BODY]) return false;
          lexer->result_symbol = UNTERMINATED_UNSUPPORTED_BODY;
          return true;
        }
        line_start = false;
        body_started = true;
        break;
      }
      qlik_rem_comment(lexer);
    } else break;
  }
  if (!body_started && lexer->lookahead == ':') return false;
  while (!lexer->eof(lexer)) {
    int32_t c = lexer->lookahead;
    if (c == '\n') {
      qlik_advance(lexer);
      lexer->mark_end(lexer);
      line_start = true;
      previous = 0;
    } else if (line_start && qlik_space(c)) {
      qlik_advance(lexer);
    } else if (line_start) {
      if (qlik_label(lexer)) {
        if (!valid_symbols[UNTERMINATED_UNSUPPORTED_BODY]) return false;
        lexer->result_symbol = UNTERMINATED_UNSUPPORTED_BODY;
        return true;
      }
      line_start = false;
      previous = 0;
    } else if (c == ';') {
      qlik_advance(lexer);
      lexer->mark_end(lexer);
      if (!valid_symbols[UNSUPPORTED_BODY]) return false;
      lexer->result_symbol = UNSUPPORTED_BODY;
      return true;
    } else if (c == '\'' || c == '"' || c == '`' || c == '[') {
      qlik_group(lexer, c == '[' ? ']' : c);
      previous = 0;
    } else if (c == '$') {
      qlik_advance(lexer);
      if (lexer->lookahead == '(') qlik_group(lexer, ')');
      previous = 0;
    } else if (qlik_name_start(c)) {
      if (qlik_read_name(lexer)) qlik_rem_comment(lexer);
      previous = 0;
    } else {
      qlik_advance(lexer);
      if (c == '/' && previous != ':') qlik_comment(lexer);
      previous = c;
    }
  }
  // This may be empty immediately after an EOF keyword. The grammar consumes
  // the keyword first, so it still makes parser progress and cannot repeat.
  lexer->mark_end(lexer);
  if (!valid_symbols[UNTERMINATED_UNSUPPORTED_BODY]) return false;
  lexer->result_symbol = UNTERMINATED_UNSUPPORTED_BODY;
  return true;
}
