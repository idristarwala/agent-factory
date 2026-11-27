# Record attributes

Factory records carry an `attributes` object. A factory definition selects the fields the worker may edit and their JSON types. A source mapping can copy differently named or nested source values into those fields. Additional source attributes can supply context without becoming editable. [Example records](../examples/service-desk/records.json) · [Source mapping](MAPPING.md) · [Worker schema](SCHEMA.md)

## Mapping a source record

Profile a representative payload to inspect collection paths, candidate identifiers and observed fields. Preview the mapping to see the record ID and attributes for each row before approving a revision. For example, a service desk can map `/ticket/summary` into `title` and include `/ticket/team` as context. Selecting only `title` as editable keeps the team available to the worker while limiting what it can change. [Mapping workflow](MAPPING.md)
