# Ralph vs Oneshot: Capability Eval Results

*January 2026*

## Conclusion

**Ralph shows marginal quality improvement over oneshot, passing 2-13% more tasks across evaluation suites.** However, this comes at 1.5-5x higher cost. The context resets between tasks are not the primary value driver. The more important factor is **task decomposition and explicit test criteria** in the prompt engineering, which can be applied to any agent harness.

| Suite | Ralph | Oneshot | Δ Tasks Passed |
|-------|-------|---------|----------------|
| workflow | 85% (41/48) | 83% (40/48) | +2.5% |
| tasktracker | 93% (26/28) | 82% (23/28) | +13% |

## Methodology

### Agent Harnesses Under Test

We compare two agent harnesses that orchestrate Claude to complete coding tasks:

**Oneshot harness**: Single Claude session with full requirements. The agent generates the entire codebase in one continuous trajectory with no interruption.

**Ralph harness**: Breaks work into discrete tasks via a PRD. Each task runs in a fresh Claude session with isolated context. Tasks execute sequentially with context reset between each.

### Experimental Design

The goal was to isolate the effect of **context resets** between task completions. Both harnesses use:

- **Same model**: Claude Sonnet
- **Same task specifications**: Identical requirements.md files
- **Same graders**: Shared deterministic test suite run against both outcomes
- **Same prompts**: Prompts are as close as possible between harnesses

The only difference: **Ralph resets context between tasks**, while oneshot operates in a single continuous session.

### Evaluation Suites

| Suite | Description | Tasks | Grader Type |
|-------|-------------|-------|-------------|
| **workflow** | CLI pipeline engine with YAML parsing, dependency graphs, conditions, retries | 48 | Deterministic (unit tests) |
| **tasktracker** | REST API with JWT auth, 4 models, ~15 endpoints | 28 | Deterministic (API tests) |

### Graders

All graders are **deterministic/code-based**: the agent's outcome (generated code) is tested by running unit tests or API tests. A task passes only if all assertions pass. This approach is natural for coding agents because software is straightforward to evaluate: does the code run and do the tests pass?

## Results

### Workflow Engine CLI

| Metric | Ralph | Oneshot |
|--------|-------|---------|
| Tasks Passed | **41/48 (85%)** | 40/48 (83%) |
| Duration | 991s | 624s |
| Tokens | 4.57M | 3.13M |
| Cost | $2.61 | $1.75 |

### Task Tracker REST API

| Metric | Ralph | Oneshot |
|--------|-------|---------|
| Tasks Passed | **26/28 (93%)** | 23/28 (82%) |
| Duration | 1546s | 298s |
| Tokens | 7.83M | 1.01M |
| Cost | $4.18 | $0.72 |

### Model Comparison: Sonnet vs Opus (Workflow Suite)

| Metric | Sonnet Ralph | Sonnet Oneshot | Opus Ralph | Opus Oneshot |
|--------|--------------|----------------|------------|--------------|
| Tasks Passed | 41/48 (85%) | 40/48 (83%) | 41/48 (85%) | 40/48 (83%) |
| Duration | 991s | 624s | 522s | 395s |
| Tokens | 4.57M | 3.13M | 3.70M | 2.15M |
| Cost | $2.61 | $1.75 | $2.96 | $1.99 |

**Finding:** Opus achieved identical pass rates to Sonnet but costs ~15% more. No quality improvement from using a larger model on this task. The 2.5% Ralph advantage over oneshot holds across both models.

## Discussion

### Marginal Capability Gains

The quality improvement from context resets alone is **marginal** (2-13%). Given the significant cost increase (1.5-5x), the loop mechanism with context resets is not the primary value driver.

### What Actually Matters

The more important contribution is the **prompt engineering** around Ralph's task structure:

1. **Task decomposition** - Breaking work into discrete, well-scoped tasks with clear success criteria
2. **Testability emphasis** - Each task includes explicit test requirements that guide implementation
3. **Incremental verification** - Structure that enables grading each piece before moving on

These patterns can be applied to any agent harness, including oneshot approaches. The loop is a convenient mechanism to enforce them, but the discipline of task breakdown and test-driven requirements matters more than context resets.

### Cost-Quality Tradeoff

Ralph's gains come at significant cost:
- **1.5x cost** for workflow (49% more expensive)
- **5x cost** for tasktracker (480% more expensive)

For most use cases, a well-structured oneshot prompt with clear task breakdown may achieve similar quality at lower cost.

### Limitations

- Small sample size (2 suites, 1 trial each)
- Same model used for both harnesses
- Deterministic graders only (no model-based or human grading)
- Results may vary with different task complexity levels

### Challenges & Open Issues

**1. Eval design tension**

Writing deterministic graders without leaking test details into the requirements is difficult. If the requirements explicitly describe how the agent will be tested, the eval becomes trivial. If they're too vague, the graders become brittle, failing on valid implementations that don't match expected output formats.

This tension is inherent to capability evals for coding agents. Our current approach uses outcome-based graders (does the code work?) rather than transcript-based graders (did the agent follow specific steps?), which helps but doesn't eliminate the problem.

**2. Task granularity tradeoff**

Ralph allows the agent to complete multiple tasks per loop iteration if it chooses. We tested forcing exactly 1 task per iteration:

- **Minimal improvement** in task pass rate
- **Major increase** in token cost (context resets on every task)

The flexibility to batch related tasks appears to be the right default. It reduces cost without sacrificing quality.

**3. Greenfield bias**

Both eval suites are greenfield projects built from scratch. Context rot and compaction are less of a problem when there's no existing codebase to navigate. In large, complex codebases where the agent must read and modify existing code across many files, Ralph's context resets might provide more benefit by preventing accumulated context from degrading quality. These evals don't capture that scenario.
