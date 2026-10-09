(set_statement name: (variable_name) @name.definition.variable) @definition.variable
(let_statement name: (variable_name) @name.definition.variable) @definition.variable
(table_label name: (table_name) @name.definition.table) @definition.table
(load_field alias: (field_name) @name.definition.field) @definition.field
(load_field !alias expression: (field_reference) @name.definition.field) @definition.field
(hierarchy_prefix parent_name: (field_name) @name.definition.field) @definition.field
(hierarchy_prefix path_name: (field_name) @name.definition.field) @definition.field
(hierarchy_prefix depth: (field_name) @name.definition.field) @definition.field
(store_statement source: (data_source) @name.definition.source) @definition.source
