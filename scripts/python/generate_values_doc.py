import re
import os
import yaml
import textwrap

def soft_break(text, min_len=20, prefer_chars=('.', '_')):
    """
    Insert a <br> after every '.' or '_' that appears after each min_len characters.
    For each segment of min_len, if a '.' or '_' is found after that point, break after it and continue.
    If no such character is found after min_len, search backward in the last min_len segment for the last '.' or '_'.
    If still not found, do not break and append the rest as is.
    """
    if not text or len(text) <= min_len:
        return text
    result = ''
    start = 0
    while start < len(text):
        if len(text) - start <= min_len:
            result += text[start:]
            break
        # Search forward for next break char after min_len
        forward_idxs = [text.find(c, start + min_len) for c in prefer_chars]
        forward_idxs = [i for i in forward_idxs if i != -1]
        if forward_idxs:
            idx = min(forward_idxs)
            result += text[start:idx+1] + '<br>'
            start = idx+1
            continue
        # Search backward in the last min_len segment for the last break char
        segment = text[start:start+min_len]
        last_idx = -1
        for c in prefer_chars:
            idx = segment.rfind(c)
            if idx > last_idx:
                last_idx = idx
        if last_idx != -1:
            result += text[start:start+last_idx+1] + '<br>'
            start = start+last_idx+1
            continue
        # No break char found, append the rest and break
        result += text[start:]
        break
    return result

def extract_block_scalars(lines):
    """
    Scan YAML lines and return a dict mapping key paths (dot notation) to their full block scalar value as a string.
    """
    block_scalars = {}
    key_stack = []
    i = 0
    while i < len(lines):
        line = lines[i]
        match = re.match(r'^(\s*)([\w\-]+):\s*([|>]\-?)', line)
        if match:
            indent, key, block_type = match.groups()
            indent_level = len(indent)
            # Build the key path
            while key_stack and key_stack[-1][1] >= indent_level:
                key_stack.pop()
            key_stack.append((key, indent_level))
            key_path = '.'.join([k for k, _ in key_stack])
            # Extract block
            block_lines = []
            i += 1
            while i < len(lines):
                next_line = lines[i]
                if not next_line.strip():
                    block_lines.append('')
                    i += 1
                    continue
                next_indent = len(next_line) - len(next_line.lstrip(' '))
                if next_indent > indent_level:
                    block_lines.append(next_line[indent_level+1:] if next_line[indent_level+1:] else '')
                    i += 1
                else:
                    break
            # Dedent and clean up block value
            block_value = '\n'.join(block_lines)
            block_value = textwrap.dedent(block_value)
            block_value = block_value.strip('\n')
            block_scalars[key_path] = block_value
            continue
        # Not a block scalar, just update stack
        match2 = re.match(r'^(\s*)([\w\-]+):', line)
        if match2:
            indent, key = match2.groups()
            indent_level = len(indent)
            while key_stack and key_stack[-1][1] >= indent_level:
                key_stack.pop()
            key_stack.append((key, indent_level))
        i += 1
    return block_scalars

def format_default_value(val):
    if val is None or val == 'None':
        return 'None'
    # Convert all types to string and wrap in backticks, escaping for Markdown
    val_str = str(val)
    val_str = val_str.replace('|', '\|')
    val_str = val_str.replace('`', '\`')
    val_str = val_str.strip('\n')
    val_str = val_str.replace('\n', '  ')
    return f'`{val_str}`'

def parse_values_yaml_with_comments(filepath):
    with open(filepath, 'r') as f:
        lines = f.readlines()
    with open(filepath, 'r') as f:
        yaml_data = yaml.safe_load(f)

    block_scalars = extract_block_scalars(lines)

    # Helper to find the line for a given key path (tracks indentation and parent keys)
    def find_line_for_key(path):
        key_stack = []  # (key, indent_level)
        for idx, line in enumerate(lines):
            match = re.match(r'^(\s*)([\w\-]+):', line)
            if not match:
                continue
            indent, key = match.groups()
            indent_level = len(indent)
            # Pop stack to current indent
            while key_stack and key_stack[-1][1] >= indent_level:
                key_stack.pop()
            key_stack.append((key, indent_level))
            current_path = [k for k, _ in key_stack]
            if current_path == path:
                return idx
        return None

    # For each key, walk upwards, skipping blank lines, and collect contiguous comment lines
    def get_comment_above(idx):
        comment_lines = []
        i = idx - 1
        while i >= 0:
            line = lines[i].rstrip('\n')
            if not line.strip():
                i -= 1
                continue  # skip blank lines
            if line.strip().startswith('#'):
                comment_lines.insert(0, line.strip().lstrip('#').strip())
                i -= 1
                continue
            break  # stop at first non-comment, non-blank line
        return ' '.join(comment_lines).strip() if comment_lines else '-'

    entries = []
    def walk(obj, path, line_hint=None):
        full_key = '.'.join(path)
        # If this key is a block scalar, use the pre-extracted block value and do not recurse
        if full_key in block_scalars:
            idx = find_line_for_key(path)
            desc = get_comment_above(idx) if idx is not None else '-'
            entries.append((full_key, desc, block_scalars[full_key]))
            return
        # If it's a string (including block scalar), treat as leaf and use as default value
        if isinstance(obj, str) or obj is None:
            idx = find_line_for_key(path)
            desc = get_comment_above(idx) if idx is not None else '-'
            entries.append((full_key, desc, obj if obj is not None else 'None'))
            return
        # Only recurse into dicts and lists, never into anything else
        if isinstance(obj, dict):
            for k, v in obj.items():
                walk(v, path + [k], line_hint)
        elif isinstance(obj, list):
            # Document lists as well
            idx = find_line_for_key(path)
            desc = get_comment_above(idx) if idx is not None else '-'
            # Convert list to YAML/Markdown-friendly string
            if len(obj) == 0:
                list_str = '[]'
            else:
                list_str = '\n'.join([f'- {str(item)}' for item in obj])
            entries.append((full_key, desc, list_str))
            return
        elif isinstance(obj, (int, float, bool)):
            idx = find_line_for_key(path)
            desc = get_comment_above(idx) if idx is not None else '-'
            entries.append((full_key, desc, obj))
        # else: skip other types
    walk(yaml_data, [])
    return entries

def generate_markdown_table(entries):
    table = "| Parameter | Description | Default Value |\n"
    table += "| --- | --- | --- |\n"
    for param, desc, default in entries:
        param_disp = soft_break(param, 20, prefer_chars=('.', '_'))
        default_disp = format_default_value(default)
        table += f"| {param_disp} | {desc if desc else '-'} | {default_disp} |\n"
    return table

if __name__ == "__main__":
    yaml_path = os.path.join("resources", "keb", "values.yaml")
    output_path = os.path.join("docs", "contributor", "02-70-chart-config.md")
    entries = parse_values_yaml_with_comments(yaml_path)
    md_table = generate_markdown_table(entries)
    md_table = re.sub(r'`?TBD`?', 'None', md_table)
    with open(output_path, 'w') as f:
        f.write(md_table)
    print(f"Documentation table written to {output_path}")
