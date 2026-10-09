(variable_reference name: (variable_name) @name.reference.variable) @reference.variable
(bare_variable_reference) @name.reference.variable @reference.variable
(field_reference) @name.reference.field @reference.field
(resident_clause table: (table_name) @name.reference.table) @reference.table
(join_prefix table: (table_name) @name.reference.table) @reference.table
(concatenate_prefix table: (table_name) @name.reference.table) @reference.table
(store_statement table: (table_name) @name.reference.table) @reference.table
(from_clause source: (data_source) @name.reference.source) @reference.source
(include_statement source: (include_path) @name.reference.include) @reference.include
