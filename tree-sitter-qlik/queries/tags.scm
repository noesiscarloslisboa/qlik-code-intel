; Aider-compatible name.definition.* / name.reference.* capture conventions.
(set_statement name: (variable_name) @name.definition.variable) @definition.variable
(let_statement name: (variable_name) @name.definition.variable) @definition.variable
(table_label name: (table_name) @name.definition.table) @definition.table
(load_field alias: (field_name) @name.definition.field) @definition.field
(load_field !alias expression: (field_reference) @name.definition.field) @definition.field
(hierarchy_prefix parent_name: (field_name) @name.definition.field) @definition.field
(hierarchy_prefix path_name: (field_name) @name.definition.field) @definition.field
(hierarchy_prefix depth: (field_name) @name.definition.field) @definition.field
(store_statement source: (data_source) @name.definition.source) @definition.source
(variable_reference name: (variable_name) @name.reference.variable) @reference.variable
(bare_variable_reference) @name.reference.variable @reference.variable
(field_reference) @name.reference.field @reference.field
(resident_clause table: (table_name) @name.reference.table) @reference.table
(join_prefix table: (table_name) @name.reference.table) @reference.table
(concatenate_prefix table: (table_name) @name.reference.table) @reference.table
(store_statement table: (table_name) @name.reference.table) @reference.table
(from_clause source: (data_source) @name.reference.source) @reference.source
(include_statement source: (include_path) @name.reference.include) @reference.include
(field_rename new: (field_name) @name.definition.field) @definition.field
(table_rename new: (table_name) @name.definition.table) @definition.table
(table_rename old: (table_name) @name.reference.table) @reference.table
(rename_using table: (table_name) @name.reference.table) @reference.table
(drop_table_list (table_name) @name.reference.table) @reference.table
