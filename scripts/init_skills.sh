#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Harness Engineering — Interactive Skill Installer
# ─────────────────────────────────────────────────────────────────────────────
# This script provisions expert domain skills from the awesome-skills catalog
# into the local workspace (./.agents/skills).
# ─────────────────────────────────────────────────────────────────────────────

set -euo pipefail

SKILLS_DIR="./.agents/skills"
MANIFEST_FILE="${SKILLS_DIR}/.antigravity-install-manifest.json"

# Mapping of menu options to skill folder names in the repository
get_skill_folders() {
  case "$1" in
    1) echo "cc-skill-clickhouse-io" ;;
    2) echo "test-driven-development" ;;
    3) echo "debugging-strategies" ;;
    4) echo "golang-pro,architecture-patterns" ;;
    *) echo "" ;;
  esac
}

# Bundle names for logging
get_bundle_name() {
  case "$1" in
    1) echo "@clickhouse-expert" ;;
    2) echo "@test-driven-development" ;;
    3) echo "@debugging-strategies" ;;
    4) echo "@go-clean-architecture" ;;
    *) echo "" ;;
  esac
}


# ── Check Prerequisites ──────────────────────────────────────────────────────
check_prereqs() {
  if ! command -v node &> /dev/null; then
    echo "❌ Error: 'node' is not installed or not in PATH." >&2
    echo "Please install Node.js (v18+) and try again." >&2
    exit 1
  fi

  if ! command -v npx &> /dev/null; then
    echo "❌ Error: 'npx' is not installed or not in PATH." >&2
    exit 1
  fi
}

# ── List Installed Skills ─────────────────────────────────────────────────────
list_installed_skills() {
  echo ""
  echo "📂 Currently Installed Skills in ${SKILLS_DIR}:"
  echo "─────────────────────────────────────────────────────────────────"

  if [ ! -f "${MANIFEST_FILE}" ]; then
    echo "⚠️  No installation manifest found. Directory might be empty."
    # Fallback to listing directories
    if [ -d "${SKILLS_DIR}" ]; then
      find "${SKILLS_DIR}" -maxdepth 1 -mindepth 1 -type d ! -name "docs" -exec basename {} \;
    else
      echo "No skills installed yet."
    fi
    return
  fi

  # Read entries from manifest using Node.js for portability
  node -e "
    try {
      const manifest = JSON.parse(require('fs').readFileSync('${MANIFEST_FILE}', 'utf8'));
      if (manifest.entries && manifest.entries.length > 0) {
        manifest.entries.forEach(e => {
          if (e !== 'docs') console.log('  • ' + e);
        });
      } else {
        console.log('No skills listed in manifest.');
      }
    } catch (err) {
      console.error('Error reading manifest:', err.message);
    }
  "
  echo "─────────────────────────────────────────────────────────────────"
}

# ── Remove a Specific Skill ───────────────────────────────────────────────────
remove_skill() {
  local target_skill=""
  if [ "${1:-}" != "" ]; then
    target_skill="$1"
  else
    list_installed_skills
    read -p "❓ Enter the name of the skill to remove: " target_skill
  fi

  target_skill=$(echo "${target_skill}" | xargs)

  if [ -z "${target_skill}" ]; then
    echo "❌ No skill specified."
    return
  fi

  local skill_path="${SKILLS_DIR}/${target_skill}"

  if [ ! -d "${skill_path}" ]; then
    echo "❌ Skill '${target_skill}' is not installed at ${skill_path}."
    return
  fi

  echo "🗑️  Removing skill '${target_skill}'..."
  rm -rf "${skill_path}"

  # Update the manifest file if it exists
  if [ -f "${MANIFEST_FILE}" ]; then
    node -e "
      const fs = require('fs');
      try {
        const manifest = JSON.parse(fs.readFileSync('${MANIFEST_FILE}', 'utf8'));
        const before = manifest.entries.length;
        manifest.entries = manifest.entries.filter(e => e !== '${target_skill}');
        manifest.updatedAt = new Date().toISOString();
        fs.writeFileSync('${MANIFEST_FILE}', JSON.stringify(manifest, null, 2) + '\n', 'utf8');
        console.log('✅ Updated manifest.');
      } catch (err) {
        console.error('⚠️  Failed to update manifest:', err.message);
      }
    "
  fi

  echo "✅ Skill '${target_skill}' successfully removed."
}

# ── Install Skills ────────────────────────────────────────────────────────────
install_skills() {
  local selection="$1"
  check_prereqs

  # Create base folders if missing
  mkdir -p "${SKILLS_DIR}"

  if [ "${selection}" == "5" ]; then
    echo "📥 Installing ALL available skills..."
    # Execute installer with path pointing to the project subfolder
    npx -y antigravity-awesome-skills install --path "${SKILLS_DIR}"
    echo "✅ Successfully installed all available skills."
    return
  fi

  # For selective installation, we clone to a temp directory,
  # copy the selected folders, update the manifest, and clean up.
  echo "📥 Preparing selective installation..."
  local temp_dir
  temp_dir=$(mktemp -d -t "ag-skills-init-XXXXXX")

  # Clone repo
  echo "Cloning awesome-skills repository..."
  git clone --depth 1 https://github.com/sickn33/antigravity-awesome-skills.git "${temp_dir}"

  # Parse the comma-separated selection
  IFS=',' read -ra ADDR <<< "${selection}"
  local installed_list=()

  for choice in "${ADDR[@]}"; do
    choice=$(echo "${choice}" | xargs)
    if [ -z "${choice}" ]; then continue; fi

    if [[ ! "${choice}" =~ ^[1-4]$ ]]; then
      echo "⚠️  Skipping invalid choice: ${choice}"
      continue
    fi

    local skill_folders
    skill_folders=$(get_skill_folders "${choice}")
    local bundle_name
    bundle_name=$(get_bundle_name "${choice}")


    echo "⚙️  Provisioning ${bundle_name}..."

    # Split if mapping has multiple folders (comma-separated)
    IFS=',' read -ra FOLDERS <<< "${skill_folders}"
    for folder in "${FOLDERS[@]}"; do
      folder=$(echo "${folder}" | xargs)
      local src_path="${temp_dir}/skills/${folder}"

      if [ ! -d "${src_path}" ]; then
        # Fallback check directly in root/subfolders of cloned repo if needed
        echo "⚠️  Folder ${folder} not found in repository. Skipping."
        continue
      fi

      echo "   Copying ${folder} to local workspace..."
      cp -R "${src_path}" "${SKILLS_DIR}/${folder}"
      installed_list+=("${folder}")
    done
  done

  # Copy docs and manifest baseline if present
  if [ -d "${temp_dir}/docs" ]; then
    mkdir -p "${SKILLS_DIR}/docs"
    cp -R "${temp_dir}/docs/" "${SKILLS_DIR}/docs/"
    installed_list+=("docs")
  fi

  # Clean up temp clone
  rm -rf "${temp_dir}"

  # Update the manifest file
  if [ ${#installed_list[@]} -gt 0 ]; then
    node -e "
      const fs = require('fs');
      const manifestPath = process.argv[1];
      const newEntries = process.argv.slice(2);
      let manifest = { schemaVersion: 1, updatedAt: '', entries: [] };
      if (fs.existsSync(manifestPath)) {
        try {
          manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
        } catch (e) {}
      }
      
      const combined = [...new Set([...manifest.entries, ...newEntries])].sort();
      manifest.entries = combined;
      manifest.updatedAt = new Date().toISOString();
      
      fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + '\n', 'utf8');
    " "${MANIFEST_FILE}" "${installed_list[@]}"
  fi

  echo "✅ Provisioning complete."
}

# ── Main Loop / CLI Entrypoint ───────────────────────────────────────────────
main() {
  # Handle command line flags first
  if [ "${1:-}" != "" ]; then
    case "$1" in
      --list|-l)
        list_installed_skills
        exit 0
        ;;
      --remove|-r)
        if [ "${2:-}" == "" ]; then
          echo "❌ Error: Please specify the skill folder name to remove." >&2
          exit 1
        fi
        remove_skill "$2"
        exit 0
        ;;
      --install|-i)
        if [ "${2:-}" == "" ]; then
          echo "❌ Error: Please specify the selection (e.g. '1,3' or '5')." >&2
          exit 1
        fi
        install_skills "$2"
        exit 0
        ;;
      --help|-h)
        echo "Usage: $0 [options]"
        echo "Options:"
        echo "  -i, --install <selection>  Install specific bundles (e.g., '1,3' or '5')"
        echo "  -l, --list                 List currently installed skills"
        echo "  -r, --remove <name>        Remove a specific skill"
        echo "  -h, --help                 Show this help message"
        exit 0
        ;;
      *)
        echo "❌ Unknown option: $1" >&2
        exit 1
        ;;
    esac
  fi

  # Interactive Menu Mode
  check_prereqs
  clear || true
  echo "========================================================"
  echo "      Harness Engineering — Expert Skill Installer      "
  echo "========================================================"
  echo "Choose which expert domain skills to provision:"
  echo ""
  echo "  1) @clickhouse-expert         (cc-skill-clickhouse-io)"
  echo "  2) @test-driven-development   (test-driven-development)"
  echo "  3) @debugging-strategies       (debugging-strategies)"
  echo "  4) @go-clean-architecture     (golang-pro + patterns)"
  echo "  5) Install ALL available skills (Advanced/Full setup)"
  echo "  6) List currently installed skills"
  echo "  7) Remove a specific skill"
  echo "  8) Exit / Cancel"
  echo ""
  echo "========================================================"
  
  read -p "❓ Enter choices (comma-separated, e.g. 1,3 or 5): " user_input
  user_input=$(echo "${user_input}" | xargs)

  if [ -z "${user_input}" ] || [ "${user_input}" == "8" ]; then
    echo "👋 Exiting. No changes made."
    exit 0
  fi

  if [ "${user_input}" == "6" ]; then
    list_installed_skills
    exit 0
  fi

  if [ "${user_input}" == "7" ]; then
    remove_skill ""
    exit 0
  fi

  install_skills "${user_input}"
}

main "$@"
