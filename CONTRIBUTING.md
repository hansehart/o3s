# Contributing

## Changes

- **Branches** ideally start from an issue, carry one change each, and land on main through a pull request as one squashed commit.
- **Commits** start with the kind of change and a scope from `.commitlintrc.json`, followed by a short summary.
- **Descriptions** of pull requests and issues stay slim and minimal, about 2–3 sentences on average.
- **Updates** of dependencies arrive from the Renovate bot, one pull request per update.

## Code

- **Comments** state a step's intent, in abstract terms, so they stay accurate as the concrete details change. Every branch and loop carries one, on the line above it.
- **Tests** come first, so a change starts with a test that fails without it, and the implementation then makes it pass.

## Releases

- **Versions** are left to the release-please bot, whose pull request raises the first number for breaking changes, the second for new features, and the third for fixes.
- **Releases** go out when that pull request merges, and publish the artifacts the repository standards name.

## Tests

- **Layers** run from many unit tests next to their code, through integration tests in `test/integration` that raise one component, to a few e2e tests in `test/e2e` that raise the whole stack, each behind its build tag.
- **Names** state the behavior under test, and a comment above each test repeats it as a sentence.
- **Bodies** check one behavior each, in setup, action and check set apart by blank lines.
- **Checks** use the standard library and go-cmp, compare whole values, test errors by identity, and report what they got against what they wanted.
- **Helpers** set up a named state and undo it once the test ends, and fakes live where they are used, held to the real adapter by a contract test.

## Artifacts

- **Images** reach ghcr.io with every release, tagged with their version.
