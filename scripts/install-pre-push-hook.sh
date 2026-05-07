#!/usr/bin/env sh
set -eu

repo_root=$(git rev-parse --show-toplevel)
hook_path=$(git rev-parse --git-path hooks/pre-push)
script_path="$repo_root/scripts/pre-push-make-test.sh"

if [ ! -f "$script_path" ]; then
  echo "Cannot find $script_path" >&2
  exit 1
fi

mkdir -p "$(dirname "$hook_path")"
cat > "$hook_path" <<'HOOK'
#!/usr/bin/env sh
set -eu

repo_root=$(git rev-parse --show-toplevel)
exec "$repo_root/scripts/pre-push-make-test.sh" "$@"
HOOK

chmod +x "$hook_path" "$script_path"
echo "Installed Git pre-push hook at $hook_path"
