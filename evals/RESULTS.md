# Ralph vs Oneshot: Eval Results

*January 2026*

## Conclusion

**Ralph produces higher quality code than oneshot approaches, passing 2.5-13% more tests across evaluation suites.** The improvement comes at a cost tradeoff: Ralph uses 1.5-5x more tokens and takes 1.5-5x longer. For projects where correctness matters more than cost, Ralph is the better choice.

| Suite | Ralph | Oneshot | Improvement |
|-------|-------|---------|-------------|
| workflow | 85% (41/48) | 83% (40/48) | +2.5% |
| tasktracker | 93% (26/28) | 82% (23/28) | +13% |

## Methodology

### Experimental Design

The goal was to isolate the effect of **context resets** between task completions. Both approaches use:

- **Same model**: Claude Sonnet
- **Same requirements**: Identical requirements.md files
- **Same test suite**: Shared tests run against both outputs
- **Same prompts**: Prompts are as close as possible between approaches

The only difference: **Ralph resets context between tasks**, while oneshot operates in a single continuous session.

### Approaches

**Oneshot**: Single Claude session with full requirements. Claude generates the entire codebase in one pass with no interruption.

**Ralph**: Breaks work into discrete tasks via a PRD. Each task runs in a fresh Claude session with only task-specific context. Tasks execute sequentially with context reset between each.

### Evaluation Suites

| Suite | Description | Tests | Complexity |
|-------|-------------|-------|------------|
| **workflow** | CLI pipeline engine with YAML parsing, dependency graphs, conditions, retries | 48 | Medium |
| **tasktracker** | REST API with JWT auth, 4 models, ~15 endpoints | 28 | High |

## Results

### Workflow Engine CLI

| Metric | Ralph | Oneshot |
|--------|-------|---------|
| Tests Passed | **41/48 (85%)** | 40/48 (83%) |
| Duration | 991s | 624s |
| Tokens | 4.57M | 3.13M |
| Cost | $2.61 | $1.75 |

### Task Tracker REST API

| Metric | Ralph | Oneshot |
|--------|-------|---------|
| Tests Passed | **26/28 (93%)** | 23/28 (82%) |
| Duration | 1546s | 298s |
| Tokens | 7.83M | 1.01M |
| Cost | $4.18 | $0.72 |

## Discussion

### Marginal Quality Gains

The quality improvement from context resets alone is **marginal** (2-13%). Given the significant cost increase (1.5-5x), the loop mechanism with context resets is not the primary value driver.

### What Actually Matters

The more important contribution is the **prompt engineering** around Ralph's task structure:

1. **Task decomposition** - Breaking work into discrete, well-scoped tasks with clear acceptance criteria
2. **Testability emphasis** - Each task includes explicit test requirements that guide implementation
3. **Incremental verification** - Structure that enables checking each piece before moving on

These patterns can be applied to any agent framework, including oneshot approaches. The loop is a convenient mechanism to enforce them, but the discipline of task breakdown and test-driven requirements matters more than context resets.

### Cost-Quality Tradeoff

Ralph's gains come at significant cost:
- **1.5x cost** for workflow (49% more expensive)
- **5x cost** for tasktracker (480% more expensive)

For most use cases, a well-structured oneshot prompt with clear task breakdown may achieve similar quality at lower cost.

### Limitations

- Small sample size (2 suites, 1 run each)
- Same model used for both approaches
- Test suites may not capture all quality dimensions
- Results may vary with different requirement complexity levels
