// Original grammar. See README.md for language references and scope.
const kw = word => new RegExp(word.split('').map(c => /[a-z]/i.test(c) ? `[${c.toLowerCase()}${c.toUpperCase()}]` : c).join(''));
const commaSep1 = rule => seq(rule, repeat(seq(',', rule)));
const functionCall = ($, expression) => seq(
  field('function', choice($.identifier, $.interpretation_function)), '(',
  optional(choice(expression, $.wildcard)),
  repeat(seq(',', optional(choice(expression, $.wildcard)))), ')'
);
const traceQuoted = ($, open, close, content, escaped) => seq(open, repeat(choice(
  alias($._trace_variable_reference, $.variable_reference), token.immediate(content),
  ...(escaped ? [token.immediate(escaped)] : []), token.immediate('$')
)), close);
// Optional positional parameters retain their comma even when the value is omitted.
const optionalCommaSuffix = (...rules) => {
  const [first, ...remaining] = rules;
  return optional(seq(',', optional(first), ...(remaining.length ? [optionalCommaSuffix(...remaining)] : [])));
};
const binaryExpression = ($, expression) => choice(...[
  [1, choice(kw('OR'), kw('XOR'))], [2, kw('AND')],
  [3, choice('=', '<>', '!=', '<', '>', '<=', '>=', kw('LIKE'))],
  [4, '&'], [5, choice('+', '-')], [6, choice('*', '/')], [7, '^']
].map(([p, op]) => prec.left(p, seq(field('left', expression), field('operator', op), field('right', expression)))));

module.exports = grammar({
  name: 'qlik',
  extras: $ => [/\s/, /\uFEFF/, $.comment],
  word: $ => $.identifier,
  externals: $ => [$.unsupported_body, $.unterminated_unsupported_body, $.name_fragment, $._immediate_name_fragment, $._error_sentinel],

  rules: {
    source_file: $ => repeat(choice(
      $.set_statement, $.let_statement, $.load_statement, $.store_statement,
      $.include_statement, $.rename_field_statement, $.rename_table_statement,
      $.drop_field_statement, $.drop_table_statement, $.trace_statement,
      $.unsupported_control_statement, $.unsupported_statement,
      $.unsupported_literal_statement, ';'
    )),
    comment: _ => token(choice(
      seq('//', /[^\r\n]*/),
      seq('/*', /[^*]*\*+([^/*][^*]*\*+)*/, '/'),
      seq(kw('REM'), choice(';', seq(/[ \t\r\n]+/, /[^;]*/, ';')))
    )),

    set_statement: $ => seq(kw('SET'), field('name', $.variable_name), '=', optional(field('value', $.set_value)), ';'),
    let_statement: $ => seq(kw('LET'), field('name', $.variable_name), '=', optional(field('value', $._let_expression)), ';'),
    // SET assigns text, not an evaluated expression. Strings still protect semicolons.
    set_value: $ => repeat1(choice($.string, $.quoted_name, $.backtick_name, $.variable_reference, $.set_text, '$')),
    set_text: _ => token(prec(-1, /[^;'"`$\s][^;'"`$\r\n]*/)),
    variable_name: $ => choice($._name, $.variable_reference),
    variable_reference: $ => seq('$(', optional('#'), field('name', $.variable_name), optional(seq(',', commaSep1($._expression))), ')'),
    _immediate_variable_reference: $ => seq(token.immediate('$('), optional('#'), field('name', $.variable_name), optional(seq(',', commaSep1($._expression))), ')'),

    table_label: $ => seq(field('name', $.table_name), ':'),
    table_name: $ => choice($._name, $.variable_reference),
    load_statement: $ => seq(
      optional($.table_label), repeat($.load_prefix), kw('LOAD'), optional(kw('DISTINCT')),
      field('fields', $.field_list),
      optional(choice($.from_clause, $.resident_clause, $.autogenerate_clause, $.inline_clause)),
      optional($.where_clause), optional($.while_clause), optional($.group_by_clause), optional($.order_by_clause),
      ';'
    ),
    load_prefix: $ => choice($.join_prefix, $.concatenate_prefix, $.hierarchy_prefix, $.bundle_prefix, kw('NOCONCATENATE'), kw('MAPPING')),
    join_prefix: $ => seq(optional(choice(kw('LEFT'), kw('RIGHT'), kw('INNER'), kw('OUTER'))), kw('JOIN'), optional(seq('(', field('table', $.table_name), ')'))),
    concatenate_prefix: $ => seq(kw('CONCATENATE'), optional(seq('(', field('table', $.table_name), ')'))),
    bundle_prefix: _ => seq(kw('BUNDLE'), optional(kw('INFO'))),
    hierarchy_prefix: $ => seq(
      kw('HIERARCHY'), '(',
      field('node_id', $._hierarchy_input), ',',
      field('parent_id', $._hierarchy_input), ',',
      field('node_name', $._hierarchy_input),
      optionalCommaSuffix(
        field('parent_name', $._hierarchy_output),
        field('path_source', $._hierarchy_input),
        field('path_name', $._hierarchy_output),
        field('path_delimiter', choice($.string, $._name, $.variable_reference)),
        field('depth', $._hierarchy_output)
      ), ')'
    ),
    _hierarchy_input: $ => choice($.field_reference, $.variable_reference),
    _hierarchy_output: $ => choice($.field_name, alias($.string, $.field_name)),
    field_list: $ => commaSep1(choice($.wildcard, $.load_field)),
    wildcard: _ => '*',
    load_field: $ => seq(field('expression', $._expression), optional(seq(kw('AS'), field('alias', $.field_name)))),
    field_name: $ => choice($._name, $.variable_reference),
    field_reference: $ => $._name,
    from_clause: $ => seq(kw('FROM'), field('source', $.data_source), optional($.format_spec)),
    resident_clause: $ => seq(kw('RESIDENT'), field('table', $.table_name)),
    autogenerate_clause: $ => seq(kw('AUTOGENERATE'), $._expression),
    inline_clause: $ => seq(kw('INLINE'), field('data', $.bracket_name), optional($.format_spec)),
    where_clause: $ => seq(kw('WHERE'), $._expression),
    while_clause: $ => seq(kw('WHILE'), $._expression),
    group_by_clause: $ => seq(kw('GROUP'), kw('BY'), commaSep1($._expression)),
    order_by_clause: $ => seq(kw('ORDER'), kw('BY'), commaSep1(seq($.field_reference, optional(choice(kw('ASC'), kw('DESC')))))),
    data_source: $ => choice($.bracket_name, $.string, $.quoted_name, $.backtick_name, $.bare_path),
    bare_path: $ => seq(choice($.variable_reference, /[^\s;()\[\]'"`$]+/), repeat(choice($.variable_reference, token.immediate(/[^\s;()\[\]'"`$]+/)))),
    format_spec: $ => seq('(', repeat(choice($.format_spec, $.string, $.quoted_name, $.backtick_name, $.variable_reference, /[^()'"`$]+/, '$')), ')'),

    store_statement: $ => seq(kw('STORE'), optional(seq(field('fields', $.field_list), kw('FROM'))), field('table', $.table_name), kw('INTO'), field('source', $.data_source), optional($.format_spec), ';'),
    include_statement: $ => prec.right(seq('$(', field('mode', $.include_mode), '=', field('source', $.include_path), ')', optional(';'))),
    include_mode: _ => choice(kw('INCLUDE'), kw('MUST_INCLUDE')),
    include_path: $ => repeat1(choice($.variable_reference, $.string, $.quoted_name, $.backtick_name, $.bracket_name, /[^)$'"`\[\]\r\n]+/)),

    // Operations preserve explicit source names, without resolving a runtime
    // data model. Single quotes are name delimiters in these statement contexts.
    rename_field_statement: $ => seq(kw('RENAME'), choice(kw('FIELD'), kw('FIELDS')),
      choice($.rename_using, commaSep1($.field_rename)), ';'),
    rename_table_statement: $ => seq(kw('RENAME'), choice(kw('TABLE'), kw('TABLES')),
      choice($.rename_using, commaSep1($.table_rename)), ';'),
    rename_using: $ => seq(kw('USING'), field('table', $._operation_table_name)),
    field_rename: $ => seq(field('old', $._operation_field_reference), kw('TO'),
      field('new', choice($.field_name, alias($.string, $.field_name)))),
    table_rename: $ => seq(field('old', $._operation_table_name), kw('TO'),
      field('new', $._operation_table_name)),
    drop_field_statement: $ => seq(kw('DROP'), choice(kw('FIELD'), kw('FIELDS')),
      $.drop_field_list, optional($.drop_from), ';'),
    drop_table_statement: $ => seq(kw('DROP'), optional(field('mapping', $.mapping_keyword)),
      choice(kw('TABLE'), kw('TABLES')), $.drop_table_list, ';'),
    mapping_keyword: _ => kw('MAPPING'),
    drop_field_list: $ => commaSep1($._operation_field_reference),
    drop_table_list: $ => commaSep1($._operation_table_name),
    drop_from: $ => seq(kw('FROM'), $.drop_table_list),
    _operation_field_reference: $ => choice(alias($.field_name, $.field_reference), alias($.string, $.field_reference)),
    _operation_table_name: $ => choice($.table_name, alias($.string, $.table_name)),

    trace_statement: $ => seq(kw('TRACE'), optional($.trace_value), ';'),
    trace_value: $ => repeat1(choice($._trace_literal,
      alias($._trace_variable_reference, $.variable_reference), $.trace_text, '$', '/')),
    trace_text: _ => token(prec(-1, /[^;'"`$\[\]\/\s][^;'"`$\[\]\/\r\n]*/)),
    // Macro arguments in TRACE are text, including function-like fragments.
    // Retain nested explicit expansions without creating LOAD field nodes.
    _trace_variable_reference: $ => seq('$(', optional('#'), field('name', $.variable_name),
      optional(seq(',', optional($.trace_argument), repeat(seq(',', optional($.trace_argument))))), ')'),
    trace_argument: $ => repeat1(choice($._trace_literal, $.trace_argument_group,
      alias($._trace_variable_reference, $.variable_reference), $.trace_argument_text, '$', '/')),
    trace_argument_group: $ => seq('(', optional($.trace_argument), repeat(seq(',', optional($.trace_argument))), ')'),
    trace_argument_text: _ => token(prec(-1, /[^;'"`$\[\]()\/,\s][^;'"`$\[\]()\/,\r\n]*/)),
    _trace_literal: $ => choice(alias($._trace_string, $.string), alias($._trace_quoted_name, $.quoted_name),
      alias($._trace_bracket_name, $.bracket_name), alias($._trace_backtick_name, $.backtick_name)),
    _trace_string: $ => traceQuoted($, "'", "'", /[^'$]+/, "''"),
    _trace_quoted_name: $ => traceQuoted($, '"', '"', /[^"$]+/, '""'),
    _trace_bracket_name: $ => traceQuoted($, '[', ']', /[^\]$]+/, '$$'),
    _trace_backtick_name: $ => traceQuoted($, '`', '`', /[^`$]+/),

    _name: $ => choice($.identifier, $.bracket_name, $.quoted_name, $.backtick_name, $.expanded_name),
    expanded_name: $ => choice(
      seq($.name_fragment, $.variable_reference, repeat($._name_continuation)),
      seq($.variable_reference, repeat1($._name_continuation))
    ),
    _name_continuation: $ => choice(
      seq(alias($._immediate_name_fragment, $.name_fragment), $.variable_reference),
      alias($._immediate_variable_reference, $.variable_reference),
      $._name_suffix
    ),
    _name_suffix: _ => token.immediate(/[%@A-Za-z0-9_.$\u00c0-\uffff]+/),
    identifier: _ => /[%@A-Za-z_\u00c0-\uffff][%@A-Za-z0-9_.$\u00c0-\uffff]*/,
    bracket_name: $ => seq('[', repeat(choice($.variable_reference, token.immediate(/[^\]$]+/), token.immediate('$$'), token.immediate('$'))), ']'),
    quoted_name: $ => seq('"', repeat(choice($.variable_reference, token.immediate(/[^"$]+/), token.immediate('""'), token.immediate('$'))), '"'),
    backtick_name: $ => seq('`', repeat(choice($.variable_reference, token.immediate(/[^`$]+/), token.immediate('$'))), '`'),
    string: $ => seq("'", repeat(choice($.variable_reference, token.immediate(/[^'$]+/), token.immediate("''"), token.immediate('$'))), "'"),
    number: _ => /(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?/,
    _expression: $ => choice($.number, $.string, $.field_reference, $.variable_reference, $.function_call, $.parenthesized_expression, $.unary_expression, $.binary_expression),
    interpretation_function: _ => /[A-Za-z_\u00c0-\uffff][A-Za-z0-9_\u00c0-\uffff]*#/,
    function_call: $ => functionCall($, $._expression),
    parenthesized_expression: $ => seq('(', $._expression, ')'),
    unary_expression: $ => prec(9, seq(choice('+', '-', token(prec(1, kw('NOT')))), $._expression)),
    binary_expression: $ => binaryExpression($, $._expression),

    // Bare identifiers in a LET value refer to variables, rather than LOAD
    // fields. Alias expression containers to retain the public expression AST.
    bare_variable_reference: $ => $._name,
    _let_expression: $ => choice(
      $.number, $.string, $.bare_variable_reference, $.variable_reference,
      alias($._let_function_call, $.function_call),
      alias($._let_parenthesized_expression, $.parenthesized_expression),
      alias($._let_unary_expression, $.unary_expression),
      alias($._let_binary_expression, $.binary_expression)
    ),
    _let_function_call: $ => functionCall($, $._let_expression),
    _let_parenthesized_expression: $ => seq('(', $._let_expression, ')'),
    _let_unary_expression: $ => prec(9, seq(choice('+', '-', token(prec(1, kw('NOT')))), $._let_expression)),
    _let_binary_expression: $ => binaryExpression($, $._let_expression),

    // Balanced stray literals are invalid statements. Bound their recovery so
    // subsequent supported statements remain visible, without inventing meaning.
    unsupported_literal_statement: $ => prec.right(seq(choice($.string, $.quoted_name, $.backtick_name), optional(';'))),

    // Control headers are line-terminated in Qlik. Keeping them separate avoids
    // consuming a supported LOAD in the body while making no execution claims.
    unsupported_control_statement: $ => prec.right(seq(
      field('keyword', $.control_keyword),
      optional(token.immediate(/[ \t]+[^\r\n;]*/)), optional(';')
    )),
    control_keyword: _ => token(prec(1, choice(...['FOR', 'IF', 'ELSEIF', 'ELSE', 'ENDIF', 'NEXT', 'DO', 'LOOP', 'SUB', 'END'].map(kw)))),

    // The scanner keeps unsupported text opaque and bounds missing-semicolon
    // recovery at the next unquoted, line-start table label.
    unsupported_statement: $ => prec(-10, seq(field('keyword', $.identifier), choice($.unsupported_body, $.unterminated_unsupported_body))),
  }
});
