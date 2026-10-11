# S018 data model

Command metadata declares paths/aliases, operand bounds, option arity/repeatability/exclusivity, help defaults/limits/examples and input schema identity. The pure parse result holds selected workspace/request identity, machine/help election and normalized command tokens.

Response retains current runtime-response fields; result and error may coexist. Successful responses use stdout; failed responses use stderr. Exact integer tokens remain numeric.

Date import options carry whether captured timezone is an implicit default; per-item explicit choices clear that marker. Existing Date observations remain immutable.

Work keeps existing fields/proof shape. Error accepts a bounded known generated code; old operation_failed stays valid. Results expose current cause and per-item context. Terminal/retry/lease transitions remain unchanged.

Document-version compatibility permits historical 1.0.0 persisted input under its exact schema. IPC version remains the installed candidate version.
