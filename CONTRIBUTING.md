# Contributing

## Commit Message Rules

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) with a prefix.

Format:

```text
<type>(<scope>): <subject>
```

Examples:

```text
feat(cli): add worktree open command
fix(path): handle spaces in repository path
docs(readme): update usage examples
chore(ci): add Go test workflow
```

### Rules

1. Write commit messages in English.
2. Always use a prefix (`type`), and prefer including a scope.
3. Keep the subject short and specific (roughly up to 50 characters when possible).
4. Use imperative mood in the subject (for example: `add`, `fix`, `update`).
5. Do not end the subject with a period.
6. If needed, add a body separated by a blank line.
7. Wrap body lines around 72 characters.
8. Explain what and why in the body; avoid repeating the subject.

### Allowed Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Formatting changes (no logic change)
- `refactor`: Code changes without behavior change
- `perf`: Performance improvements
- `test`: Add or update tests
- `build`: Build system or dependency changes
- `ci`: CI/CD configuration changes
- `chore`: Maintenance work
- `revert`: Revert a previous commit

