#!/usr/bin/env python3
import re
import yaml
import io

# This script generates Markdown documentation for environment variables defined in a Helm deployment YAML and values.yaml.
# It extracts environment variables from the deployment template, maps them to their descriptions and default values from values.yaml (including comments),
# and outputs a Markdown table. It handles static, single, and composite value references robustly.


# Custom loader to load all scalars as strings (including booleans)
class StrLoader(yaml.SafeLoader):
    pass

def str_constructor(loader, node):
    return loader.construct_scalar(node)

# Remove boolean resolver so 'true'/'false' are loaded as strings
for bool_tag in ['bool', 'bool#yes', 'bool#no']:
    if bool_tag in yaml.SafeLoader.yaml_implicit_resolvers:
        del yaml.SafeLoader.yaml_implicit_resolvers[bool_tag]
StrLoader.add_constructor('tag:yaml.org,2002:bool', str_constructor)
StrLoader.add_constructor('tag:yaml.org,2002:str', str_constructor)

DEPLOYMENT_YAML = "resources/keb/templates/deployment.yaml"
SUBACC_CLEANUP_YAML = "resources/keb/templates/subaccount-cleanup-job.yaml"
VALUES_YAML = "resources/keb/values.yaml"
OUTPUT_MD = "docs/contributor/02-30-keb-configuration.md"
SUBACC_MD = "docs/contributor/06-30-subaccount-cleanup-cronjob.md"
TRIAL_CLEANUP_YAML = "resources/keb/templates/trial-cleanup-job.yaml"
FREE_CLEANUP_YAML = "resources/keb/templates/free-cleanup-job.yaml"
TRIAL_FREE_MD = "docs/contributor/06-40-trial-free-cleanup-cronjobs.md"
DEPROV_RETRIGGER_YAML = "resources/keb/templates/deprovision-retrigger-job.yaml"
DEPROV_RETRIGGER_MD = "docs/contributor/06-50-deprovision-retrigger-cronjob.md"
ARCHIVER_YAML = "utils/archiver/kyma-environment-broker-archiver.yaml"
ARCHIVER_MD = "docs/contributor/06-60-archiver-job.md"
SERVICE_BINDING_CLEANUP_YAML = "resources/keb/templates/service-binding-cleanup-job.yaml"
SERVICE_BINDING_CLEANUP_MD = "docs/contributor/06-70-service-binding-cleanup-cronjob.md"
RUNTIME_RECONCILER_YAML = "resources/keb/templates/runtime-reconciler-deployment.yaml"
RUNTIME_RECONCILER_MD = "docs/contributor/07-10-runtime-reconciler.md"
SUBACCOUNT_SYNC_YAML = "resources/keb/templates/subaccount-sync-deployment.yaml"
SUBACCOUNT_SYNC_MD = "docs/contributor/07-20-subaccount-sync.md"
SCHEMA_MIGRATOR_YAML = "resources/keb/templates/migrator-job.yaml"
SCHEMA_MIGRATOR_MD = "docs/contributor/07-30-schema-migrator.md"

def extract_env_vars_with_paths(deployment_yaml_path):
    """
    Extract environment variables and their value sources from a Helm deployment YAML.
    Handles static values, single .Values references, and composite values (multiple .Values references in one value).
    Returns a list of tuples: (env_var_name, value_path_or_literal)
    """
    env_vars = []
    with open(deployment_yaml_path, "r") as f:
        lines = f.readlines()
    in_env = False
    current_env = None
    for i, line in enumerate(lines):
        if re.match(r"\s*env:\s*$", line):
            in_env = True
            continue
        if in_env:
            m = re.match(r"\s*-\s*name:\s*([A-Z0-9_]+)", line)
            if m:
                current_env = m.group(1)
                # Look ahead for value line
                for j in range(i+1, min(i+2, len(lines))):
                    val_line = lines[j]
                    # Check for composite value with multiple .Values references
                    if re.search(r'{{.*\.Values\..*}}.*{{.*\.Values\..*}}', val_line):
                        # Extract the whole value string inside 'value:'
                        mval = re.search(r'value:\s*"?(.+?)"?$', val_line)
                        if mval:
                            env_vars.append((current_env, mval.group(1).strip()))
                            break
                    else:
                        mval = re.search(r'value:\s*"?{{\s*\.Values\.([^"}}]+)\s*}}"?', val_line)
                        if mval:
                            env_vars.append((current_env, mval.group(1)))
                            break
                        elif 'valueFrom:' in val_line:
                            env_vars.append((current_env, None))
                            break
                else:
                    env_vars.append((current_env, None))
            elif re.match(r"\s*-\s*name:", line):
                continue
            elif re.match(r"\s*ports:\s*", line):
                in_env = False
    return env_vars

def parse_values_yaml_with_comments(values_yaml_path):
    """
    Parse values.yaml, extracting a mapping from dot-paths to (description, default value).
    Comments immediately preceding a key are used as the description for that key.
    Returns a dict: {dot.path: {description: str, default: value}}
    """
    with open(values_yaml_path, "r") as f:
        lines = f.readlines()
    # Use custom loader to keep all values as strings
    yaml_data = yaml.load(open(values_yaml_path), Loader=StrLoader)
    doc_map = {}
    path_stack = []
    comment_accumulator = []
    for idx, line in enumerate(lines):
        # Comments: accumulate until next key
        if line.strip().startswith('#'):
            comment_accumulator.append(line.strip('# ').strip())
            continue
        # Key
        m = re.match(r'^(\s*)([a-zA-Z0-9_\-]+):', line)
        if m:
            indent, key = len(m.group(1)), m.group(2)
            # Maintain stack for nested keys
            while path_stack and path_stack[-1][0] >= indent:
                path_stack.pop()
            path_stack.append((indent, key))
            # Compose full key path
            full_key = ".".join([k for _, k in path_stack])
            # Get value from yaml_data
            try:
                val = yaml_data
                for k in full_key.split('.'):
                    val = val[k]
                # If value is a dict, skip (not a leaf)
                if isinstance(val, dict):
                    value = ''
                else:
                    value = val
            except Exception:
                value = ''
            doc_map[full_key] = {
                'description': ' '.join(comment_accumulator).strip(),
                'default': value
            }
            comment_accumulator = []  # Reset after assigning to a key
        # If not a key or comment, do not reset accumulator
    return doc_map

def normalize_path(path):
    """
    Normalize a YAML path for matching: replace _ and - with . and lowercase.
    Does not split camelCase.
    """
    if not path:
        return ''
    return path.replace('_', '.').replace('-', '.').lower()

def map_env_to_values(env_vars, values_doc):
    """
    Map extracted environment variables to their descriptions and default values from values.yaml.
    Handles:
      - Static values (not mapped to .Values): description and value are '-'.
      - Single .Values references: use description and default from values.yaml if available.
      - Composite values (multiple .Values references): join descriptions and defaults from all referenced keys.
    Returns a list of dicts: {env, description, default}
    """
    result = []
    norm_doc_map = {normalize_path(k): v for k, v in values_doc.items()}
    for env, path in env_vars:
        desc = ''
        default = ''
        path = path.strip() if path else ''
        doc_entry = None
        norm_path = normalize_path(path) if path else ''
        # Handle composite values like '{{ .Values.host }}.{{ .Values.global.ingress.domainName }}'
        if path and re.search(r'\{\{\s*\.Values\.', path) and '}}.{{' in path:
            # Extract all .Values paths in the composite
            parts = re.findall(r'\.Values\.([a-zA-Z0-9_.]+)', path)
            descs = []
            defaults = []
            for part in parts:
                doc = values_doc.get(part) or norm_doc_map.get(normalize_path(part))
                if doc:
                    if doc.get('description', ''):
                        descs.append(doc['description'])
                    if doc.get('default', ''):
                        defaults.append(str(doc['default']))
            desc = ' / '.join(descs) if descs else '-'
            default = '.'.join(defaults) if defaults else '-'
        elif path and path in values_doc:
            doc_entry = values_doc[path]
            if doc_entry:
                desc = doc_entry.get('description', '')
                default = doc_entry.get('default', '')
        elif norm_path and norm_path in norm_doc_map:
            doc_entry = norm_doc_map[norm_path]
            if doc_entry:
                desc = doc_entry.get('description', '')
                default = doc_entry.get('default', '')
        # Otherwise, leave desc and default blank (will render as '-')
        result.append({
            'env': env,
            'description': desc,
            'default': default
        })
    return result

def soft_break(text, max_len, prefer_char=None):
    """
    Insert a soft break (\u200b) into text at max_len intervals.
    If prefer_char is set, break at the nearest prefer_char before max_len, otherwise break at max_len.
    """
    if not text or len(text) <= max_len:
        return text
    result = ''
    start = 0
    while start < len(text):
        if len(text) - start <= max_len:
            result += text[start:]
            break
        if prefer_char:
            chunk = text[start:start+max_len]
            last = chunk.rfind(prefer_char)
            if last == -1:
                # No prefer_char, just break at max_len
                result += text[start:start+max_len] + '&#x200b;'
                start += max_len
            else:
                result += text[start:start+last+1] + '&#x200b;'
                start += last+1
        else:
            result += text[start:start+max_len] + '&#x200b;'
            start += max_len
    return result

def extract_table_markdown(env_docs):
    buf = io.StringIO()
    buf.write("| Environment Variable | Current Value | Description |\n")
    buf.write("|---------------------|------------------------------|---------------------------------------------------------------|\n")
    for doc in env_docs:
        desc = doc['description'] if doc['description'] else '-'
        if doc['default'] is None or doc['default'] == '':
            default = 'None'
        else:
            default = doc["default"]
        env_val = soft_break(doc["env"], 20, prefer_char='_')
        env_col = f'**{env_val}**'
        if default == 'None':
            val_col = 'None'
        else:
            val_col = f'<code>{str(default)}</code>'
        buf.write(f"| {env_col} | {val_col} | {desc} |\n")
    return buf.getvalue()

def replace_env_table_in_md(md_path, new_table):
    with open(md_path, 'r') as f:
        lines = f.readlines()
    out = []
    in_table = False
    table_started = False
    for line in lines:
        if line.strip().startswith('| Environment variable') or line.strip().startswith('| Environment Variable'):
            in_table = True
            table_started = True
            out.append(new_table)
            continue
        if in_table:
            if not line.strip().startswith('|'):
                in_table = False
                if line.strip():
                    out.append(line)
            continue
        out.append(line)
    if not table_started:
        # If no table found, append at end
        out.append('\n' + new_table + '\n')
    with open(md_path, 'w') as f:
        f.writelines(out)

def main():
    """
    Main entry point: extract env vars, map to values.yaml, and write Markdown documentation for all jobs.
    """
    values_doc = parse_values_yaml_with_comments(VALUES_YAML)
    # KEB deployment
    env_vars = extract_env_vars_with_paths(DEPLOYMENT_YAML)
    env_docs = map_env_to_values(env_vars, values_doc)
    table = extract_table_markdown(env_docs)
    replace_env_table_in_md(OUTPUT_MD, table)
    print(f"Markdown documentation table replaced in {OUTPUT_MD}")
    # Subaccount Cleanup
    subacc_env_vars = extract_env_vars_with_paths(SUBACC_CLEANUP_YAML)
    subacc_env_docs = map_env_to_values(subacc_env_vars, values_doc)
    subacc_table = extract_table_markdown(subacc_env_docs)
    replace_env_table_in_md(SUBACC_MD, subacc_table)
    print(f"Subaccount Cleanup env documentation updated in {SUBACC_MD}")
    # Trial Cleanup
    trial_env_vars = extract_env_vars_with_paths(TRIAL_CLEANUP_YAML)
    trial_env_docs = map_env_to_values(trial_env_vars, values_doc)
    trial_table = extract_table_markdown(trial_env_docs)
    # Free Cleanup
    free_env_vars = extract_env_vars_with_paths(FREE_CLEANUP_YAML)
    free_env_docs = map_env_to_values(free_env_vars, values_doc)
    free_table = extract_table_markdown(free_env_docs)
    combined = "### Trial Cleanup CronJob\n\n" + trial_table + "\n\n### Free Cleanup CronJob\n\n" + free_table + "\n"
    replace_env_table_in_md(TRIAL_FREE_MD, combined)
    print(f"Trial/Free Cleanup env documentation updated in {TRIAL_FREE_MD}")
    # Deprovision Retrigger
    deprov_env_vars = extract_env_vars_with_paths(DEPROV_RETRIGGER_YAML)
    deprov_env_docs = map_env_to_values(deprov_env_vars, values_doc)
    deprov_table = extract_table_markdown(deprov_env_docs)
    replace_env_table_in_md(DEPROV_RETRIGGER_MD, deprov_table)
    print(f"Deprovision Retrigger env documentation updated in {DEPROV_RETRIGGER_MD}")
    # Archiver Job
    archiver_env_vars = extract_env_vars_with_paths(ARCHIVER_YAML)
    archiver_env_docs = map_env_to_values(archiver_env_vars, values_doc)
    archiver_table = extract_table_markdown(archiver_env_docs)
    replace_env_table_in_md(ARCHIVER_MD, archiver_table)
    print(f"Archiver env documentation updated in {ARCHIVER_MD}")
    # Service Binding Cleanup Job
    sbc_env_vars = extract_env_vars_with_paths(SERVICE_BINDING_CLEANUP_YAML)
    sbc_env_docs = map_env_to_values(sbc_env_vars, values_doc)
    sbc_table = extract_table_markdown(sbc_env_docs)
    replace_env_table_in_md(SERVICE_BINDING_CLEANUP_MD, sbc_table)
    print(f"Service Binding Cleanup env documentation updated in {SERVICE_BINDING_CLEANUP_MD}")
    # Runtime Reconciler
    rr_env_vars = extract_env_vars_with_paths(RUNTIME_RECONCILER_YAML)
    rr_env_docs = map_env_to_values(rr_env_vars, values_doc)
    rr_table = extract_table_markdown(rr_env_docs)
    replace_env_table_in_md(RUNTIME_RECONCILER_MD, rr_table)
    print(f"Runtime Reconciler env documentation updated in {RUNTIME_RECONCILER_MD}")
    # Subaccount Sync
    sync_env_vars = extract_env_vars_with_paths(SUBACCOUNT_SYNC_YAML)
    sync_env_docs = map_env_to_values(sync_env_vars, values_doc)
    sync_table = extract_table_markdown(sync_env_docs)
    replace_env_table_in_md(SUBACCOUNT_SYNC_MD, sync_table)
    print(f"Subaccount Sync env documentation updated in {SUBACCOUNT_SYNC_MD}")
    # Schema Migrator
    migrator_env_vars = extract_env_vars_with_paths(SCHEMA_MIGRATOR_YAML)
    migrator_env_docs = map_env_to_values(migrator_env_vars, values_doc)
    migrator_table = extract_table_markdown(migrator_env_docs)
    replace_env_table_in_md(SCHEMA_MIGRATOR_MD, migrator_table)
    print(f"Schema Migrator env documentation updated in {SCHEMA_MIGRATOR_MD}")

if __name__ == "__main__":
    main()
