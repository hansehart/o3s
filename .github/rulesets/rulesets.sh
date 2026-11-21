#!/bin/sh
# Checks and applies the rulesets the JSON files of this folder define (see README.md):
#   .github/rulesets/rulesets.sh check   fails where a live ruleset is missing or differs from its file
#   .github/rulesets/rulesets.sh apply   creates or updates the live rulesets from their files (admins only)
set -eu

repo="${GITHUB_REPOSITORY:-agentic-software-engineers/o3s}"
dir="$(dirname "$0")"

# drift prints how a live ruleset differs from its file: every value of the file must equal the live one, while the
# live ruleset may hold fields the file leaves out, such as its id. Rules compare by type, and a live rule the file
# lacks is a difference too. The bypass actors compare only where the API shows them, which it does to admins alone.
drift() {
	jq -rn --argjson want "$1" --argjson live "$2" '
		def diff($path; $w; $l):
			if ($w | type) == "object" then
				$w | keys[] as $k | diff("\($path).\($k)"; $w[$k]; ($l // {})[$k])
			elif ($w | type) == "array" then
				if ($l | type) != "array" or ($w | length) != ($l | length) then "\($path): \($w) in the file, \($l) live"
				else range($w | length) as $i | diff("\($path)[\($i)]"; $w[$i]; $l[$i]) end
			elif $w != $l then "\($path): \($w) in the file, \($l) live"
			else empty end;
		($want | del(.rules, .bypass_actors)) as $top | diff(""; $top; $live),
		(if $live.bypass_actors == null then empty else diff(".bypass_actors"; $want.bypass_actors; $live.bypass_actors) end),
		(($live.rules // []) | map({(.type): .}) | add // {}) as $byType |
		($want.rules[] | if $byType[.type] == null then ".rules.\(.type): in the file, not live" else diff(".rules.\(.type)"; .; $byType[.type]) end),
		(($live.rules // [])[] | select(.type as $t | $want.rules | map(.type) | index($t) | not)
			| ".rules.\(.type): live, not in the file")'
}

# id prints the id of the repository's own ruleset of the name given, empty where it has none.
id() {
	gh api "repos/$repo/rulesets?includes_parents=false&per_page=100" --jq ".[] | select(.name == \"$1\") | .id"
}

# check names every ruleset that is missing or differs from its file, and fails where one does.
check() {
	failed=0
	# Check every ruleset this folder defines
	for file in "$dir"/*.json; do
		name="$(jq -r .name "$file")"
		rid="$(id "$name")"
		# Name a ruleset nobody applied yet
		if [ -z "$rid" ]; then
			echo "::error file=$file::$repo has no ruleset named \"$name\". An admin applies it (see .github/rulesets/README.md)."
			failed=1
			continue
		fi
		diffs="$(drift "$(cat "$file")" "$(gh api "repos/$repo/rulesets/$rid")")"
		# Name every difference of a ruleset that drifted from its file
		if [ -n "$diffs" ]; then
			echo "::error file=$file::The ruleset \"$name\" ($rid) differs from $file. Apply the file, or update it to match the settings:%0A$(printf '%s' "$diffs" | sed ':a;N;$!ba;s/\n/%0A/g')"
			failed=1
		else
			echo "The ruleset \"$name\" ($rid) matches $file."
		fi
	done
	return "$failed"
}

# apply creates or updates every ruleset this folder defines, which takes an admin of the repository.
apply() {
	# Apply every ruleset this folder defines
	for file in "$dir"/*.json; do
		rid="$(id "$(jq -r .name "$file")")"
		# Create a ruleset the repository lacks, and update one it holds
		if [ -z "$rid" ]; then
			gh api -X POST "repos/$repo/rulesets" --input "$file" --jq '"Created \(.name) (\(.id))"'
		else
			gh api -X PUT "repos/$repo/rulesets/$rid" --input "$file" --jq '"Updated \(.name) (\(.id))"'
		fi
	done
}

# Run the command asked for
case "${1:-}" in
check) check ;;
apply) apply ;;
*)
	echo "usage: .github/rulesets/rulesets.sh check|apply" >&2
	exit 2
	;;
esac
