
# Tool Purpose
Frontmatter is a CLI tool for modifying YAML frontmatter in text files from the terminal.

# Supported Use Cases
- can add frontmatter if it doesn't exist
- can modify only specified fields without modifying others
- can return YAML frontmatter content
- can remove YAML frontmatter from file
- `--dry-run` flag shows only the changed frontmatter on stdout instead of saving changes to file

# Tool Usage

```
frontmatter [get|set|delete] [--dry-run] [...] <file>
```

# Example Usage

Simple setting of `message` field in frontmatter to value `Hello World`:
```bash
    frontmatter set message="Hello World" file.md
```

Setting nested field `object.field` to value `5`:
```bash
    frontmatter set object.field=5 file.md
```

Setting two fields `a` and `b` simultaneously:
```bash
    frontmatter set a=1 b=value file.md
```

Display value of `message` field from frontmatter to stdout:
```bash
    frontmatter get message file.md
```

Display entire frontmatter from file to stdout:
```bash
    frontmatter get file.md
``` 

Remove entire frontmatter from file:
```bash
    frontmatter delete file.md
```

Remove field from frontmatter in file:
```bash
    frontmatter delete title file.md # Remove single field
    frontmatter delete first second file.md # Remove fields first and second from file
```

Remove nested field from file:
```bash
    frontmatter delete object.field file.md
```

Getting a non-existent field should return error code 2 and not print anything to stdout.

Getting frontmatter from a file that doesn't contain frontmatter should return error code 2 and not print anything to stdout.




