# QlikView syntax coverage follow-up

The user authorized continuing the remaining parser gaps after the local app audit.
Authoritative exports remain pending; use the unchanged recovered candidates only
as local evidence. No application execution or binary-reader feature is added.

Extend the existing grammar and AST extractor with precise rules. This preserves
query captures and source ranges and permits focused regressions. An alternative
opaque-prefix rule would recover LOADs but lose argument roles and explicit field
names. A broad permissive expression token would reduce errors while weakening
retrieval accuracy. Use structured rules and the existing extractor.

Recognize empty LET values as absent expressions, preserving the assignment target
without inventing a value or references. Function calls accept omitted argument
positions represented by commas; only present expressions contribute references.
Apply the same argument rules to LOAD and LET contexts, retaining their different
field/variable reference roles. LOAD field lists still require real expressions.

Recognize BUNDLE with optional INFO before LOAD. INLINE data headers and values
produce no field/source symbols; explicit dollar expansions retain variable
references, consistent with existing strings and paths. Do not retrieve linked
images or files. SQL remains unsupported. IMAGE_SIZE and
standalone INFO are outside this follow-up.

Recognize HIERARCHY with three required input field names and up to five optional
positions. Input NodeID, ParentID, NodeName, and PathSource are field references.
Explicit ParentName, PathName, and Depth names are field definitions owned by the
LOAD table. The delimiter contributes only explicit variable expansions. Permit
omitted optional positions and quoted/composed names. Do not fabricate numbered
hierarchy levels, infer runtime transformations, or add field lineage. Definitions
and references use the existing Aider capture conventions.

Parse GROUP BY items as expressions so their component fields remain retrievable.
Qlik's LOAD documentation describes a field list, while the private candidates
contain concatenations and function calls. This parser accepts their syntax for
source retrieval without claiming those forms execute in every Qlik version or
that grouping semantics have been validated. Expression/function arity and runtime
variable values remain outside the indexer's responsibilities.

Use independently authored Go regressions, reviewed grammar corpus trees, actual
query execution, and executable CLI checks. Verify exact line/byte ranges, distinct
field roles, no references for empty values or delimiter literals, conservative
preceding LOAD boundaries, and budgeted original context. Rerun the private sample
audit, pinned external audit, and full make check. Publish aggregate changes only.

References consulted: QlikView May 2024 documentation for
[HIERARCHY](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Scripting/ScriptPrefixes/Hierarchy.htm),
[BUNDLE](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Scripting/ScriptPrefixes/Bundle.htm),
[IF](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Scripting/ConditionalFunctions/if.htm),
[LET](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Scripting/ScriptRegularStatements/Let.htm),
and [LOAD](https://help.qlik.com/en-US/qlikview/May2024/Subsystems/Client/Content/QV_QlikView/Scripting/ScriptRegularStatements/Load.htm).
Empty LET values, trailing empty arguments, and GROUP BY expressions are observed
source forms; the documentation alone does not establish all of their runtime
semantics. Keep that distinction in the assessment.
