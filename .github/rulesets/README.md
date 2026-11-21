# Rulesets

The JSON files in this folder define the repository's rulesets, in the format of GitHub's
[REST API for rulesets](https://docs.github.com/en/rest/repos/rules). GitHub doesn't read them: merging a change to one
changes nothing on the repository until an admin applies it. The [Rulesets workflow](../workflows/rulesets.yaml) checks
after every change to this folder on `main` that the live rulesets match them.

| File | Ruleset |
| --- | --- |
| [main.json](main.json) | `main`: no deletion or force push, squash pull requests, and the required checks, which hold a stable release's pull request until it is labeled `verify` and every check passes |

[rulesets.sh](rulesets.sh) compares a file with the live ruleset of the same `name`. Every value of the file must equal
the live one, while the live ruleset may hold fields the file leaves out, such as its id. Rules compare by type, and a
live rule the file lacks is a difference too. The bypass actors compare only for an admin, since the API shows them to
admins alone.

## Change a ruleset

1. Edit its file in a pull request.
2. Once it merged, an admin applies the files of an up-to-date `main`, which creates the rulesets that don't exist yet
   and updates the others:

   ```sh
   .github/rulesets/rulesets.sh apply
   ```

3. Check that the live rulesets match the files:

   ```sh
   .github/rulesets/rulesets.sh check
   ```

The Rulesets workflow fails on `main` until the change is applied. A change made in the settings goes into its file in
a pull request the same way, and `rulesets.sh check` shows whether any did.
