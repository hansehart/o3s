# Contributing

## Workflow

- **Changes** ideally start as an issue, are made on their own branch, and land on main through a pull request as one squashed commit.

## Style

- **Comments** state a step's intent, in abstract terms, so they stay accurate as the concrete details change.
- **Commits** start with the kind of change and the affected area, followed by a short summary. A version raises its first number for breaking changes, its second for new features, and its third for fixes.
- **Descriptions** of pull requests and issues stay slim and minimal, about 2–3 sentences on average.

## Tests

- **Changes** start with a failing test, followed by the least code that passes it and a refactor that keeps every test as it is.
- **Layers** run from many unit tests next to their code, through integration tests in `test/integration` that raise one component, to a few e2e tests in `test/e2e` that raise the whole stack. The build tags `integration` and `e2e` keep `go test ./...` fast.
- **Names** state the behavior under test, as in `TestTheGatewayComesUpHealthyOnAnEmptyPolicy`, and a comment above the test repeats it as a sentence.
- **Bodies** check one behavior each, in three blocks of setup, action and check, set apart by blank lines.
- **Failures** read `Func(input) = got, want want`, and larger values print `cmp.Diff(want, got)` as `(-want +got)`.
- **Helpers** set up a state and are named after it, as in `stacktest.AnEmptyPolicy`. They call `t.Helper()`, fail setup with `t.Fatalf`, and undo their work with `t.Cleanup`.
- **Checks** use the standard `testing` package and `go-cmp`, compare whole values, and test errors with `errors.Is`.
- **Doubles** are fakes defined where they are used, and one contract test holds each fake to the real adapter.
