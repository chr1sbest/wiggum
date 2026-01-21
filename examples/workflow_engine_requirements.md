# Workflow Engine CLI

Build a command-line tool that runs multi-step workflows defined in YAML files. Similar to GitHub Actions or CI pipelines but simpler.

## What It Does

Takes a YAML file with steps and runs them in order, respecting dependencies between steps. Steps can have conditions, timeouts, and can pass data to each other.

## Example Workflow

```yaml
name: deploy-app
vars:
  env: production

steps:
  - id: build
    run: go build -o app .
    
  - id: test
    run: go test ./...
    needs: [build]
    
  - id: deploy
    run: ./deploy.sh ${{ vars.env }}
    needs: [test]
    if: ${{ success() }}
    
  - id: cleanup
    run: rm -rf tmp/
    needs: [deploy]
    if: ${{ always() }}
```

## Commands

- `workflow run <file>` - Run a workflow
- `workflow validate <file>` - Check if workflow is valid (exit 0 = valid, exit 1 = invalid)
- `workflow list <file>` - Show steps in execution order

## Key Features

**Dependencies**: Steps can depend on other steps via `needs: [step1, step2]`. Steps run after their dependencies complete.

**Conditions**: Steps can have `if:` conditions using expression syntax like `${{ vars.name }}` or `${{ steps.build.outcome == 'success' }}`.

**Special condition functions**:
- `always()` - Run even if dependencies failed
- `failure()` - Run only if a dependency failed  
- `success()` - Run only if all dependencies succeeded

**Error handling**:
- `continue_on_error: true` - Don't fail the workflow if this step fails
- `timeout: 5m` - Kill step if it runs too long
- `retries: 3` - Retry failed steps

**Variables**: Define in `vars:` section, use with `${{ vars.name }}`

**Environment**: Set per-step env vars with `env:` section

**Working directory**: Change with `working_dir:`

## Execution Behavior

1. Parse the YAML and build a dependency graph
2. Detect circular dependencies (error if found)
3. Run steps in topological order
4. Evaluate `if:` conditions before running each step
5. Skip steps whose dependencies failed (unless `always()` or `failure()`)
6. Exit 0 if all steps pass, exit 1 if any fail

## Flags

- `--var key=value` - Override a workflow variable
- `--dry-run` - Show what would run without executing

## Output

Show progress as steps run. Something like:
```
[1/4] build ......... ✓
[2/4] test .......... ✓  
[3/4] deploy ........ ✓
[4/4] cleanup ....... ✓
```

If a step fails, show what failed and skip dependent steps.

## Requirements

- Language: Go
- Binary name: `workflow`
- Use gopkg.in/yaml.v3 for YAML parsing
