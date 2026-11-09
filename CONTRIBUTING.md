# Contributing

## I. Workflow

- **Changes** ideally start as an issue, are made on their own branch, and land on main through a pull request as one squashed commit.
- **Commits** start with the kind of change and its scope, followed by a short summary.
- **Descriptions** of pull requests and issues stay slim and minimal, about 2–3 sentences on average.
- **Updates** of dependencies arrive from the Renovate bot, one pull request per update.
- **Versions** are left to the release bot, whose pull request raises the first number for breaking changes, the second for new features, and the third for fixes.
- **Releases** go out when the release bot's pull request merges.

## II. Code

- **Comments** state a step's intent, in abstract terms, so they stay accurate as the concrete details change. Every branch and loop carries one, on the line above it.
- **Tests** come first, so a change starts with a test that fails without it, and the implementation then makes it pass.

## III. Releases

- **Scopes** of commits come from `.commitlintrc.json`, one per area such as `gateway` or `repo`.
- **Images** reach ghcr.io with every release, tagged with its version.

## IV. Tests

- **Refactors** follow each passing test and keep every test as it is.
- **Layers** run from many unit tests next to their code, through integration tests in `test/integration` that raise one component, to a few e2e tests in `test/e2e` that raise the whole stack. The build tags `integration` and `e2e` keep `go test ./...` fast.
- **Names** state the behavior under test, as in `TestTheGatewayComesUpHealthyOnAnEmptyPolicy`, and a comment above the test repeats it as a sentence.
- **Bodies** check one behavior each, in three blocks of setup, action and check, set apart by blank lines.
- **Failures** read `Func(input) = got, want want`, and larger values print `cmp.Diff(want, got)` as `(-want +got)`.
- **Helpers** set up a state and are named after it, as in `stacktest.AnEmptyPolicy`. They call `t.Helper()`, fail setup with `t.Fatalf`, and undo their work with `t.Cleanup`.
- **Checks** use the standard `testing` package and `go-cmp`, compare whole values, and test errors with `errors.Is`.
- **Doubles** are fakes defined where they are used, and one contract test holds each fake to the real adapter.
