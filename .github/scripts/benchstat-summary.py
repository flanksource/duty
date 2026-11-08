#!/usr/bin/env python3
"""
benchstat-summary.py — parse benchstat output and produce a markdown summary.

Usage: benchstat-summary.py [--threshold PCT] benchstat.txt

Extracts significant regressions/improvements from benchstat output.
Exit code 1 if any regression exceeds the threshold (default: 5%).
Exit code 2 if there are no comparable base/head results.
"""

import argparse
import re
import sys
from dataclasses import dataclass


@dataclass
class Change:
    name: str
    base: str
    head: str
    pct: float
    pval: str


# Matches lines like:
#   BenchmarkName-4   1.030m ±  3%   1.304m ±  5%  +26.57% (p=0.000 n=10)
#   BenchmarkName-4   3.997  ± 1%    3.910  ± 0%   -2.16%  (p=0.000 n=10)
LINE_RE = re.compile(
    r"^(\S+)"  # benchmark name
    r"\s+"
    r"(\S+)"  # base value
    r"\s+±\s+(\d+%|∞)"  # base variance
    r"\s+"
    r"(\S+)"  # head value
    r"\s+±\s+(\d+%|∞)"  # head variance
    r"\s+"
    r"([+-]\d+\.\d+%|~)"  # percentage change, or no significant change
    r"\s+"
    r"\(p=(\d+\.\d+)"  # p-value
)


def ranges_overlap(base: str, base_var: str, head: str, head_var: str) -> bool:
    """Reports whether the base and head ranges (median ± variance) overlap.

    benchstat prints both values of a row with the same unit prefix, so the numbers compare directly.
    """
    if "∞" in (base_var, head_var):
        return True
    b = float(re.sub(r"[^\d.]", "", base))
    h = float(re.sub(r"[^\d.]", "", head))
    bv = float(base_var.removesuffix("%")) / 100
    hv = float(head_var.removesuffix("%")) / 100
    return b * (1 - bv) <= h * (1 + hv) and h * (1 - hv) <= b * (1 + bv)


def parse_benchstat(path: str) -> list[Change]:
    changes = []
    comparisons = 0
    with open(path) as f:
        for line in f:
            stripped = line.strip()
            if stripped.startswith("geomean"):
                continue
            m = LINE_RE.match(stripped)
            if not m:
                continue
            comparisons += 1
            name, base, base_var, head, head_var, pct, pval = m.groups()
            # A change within the run-to-run spread of either side isn't significant, whatever its p-value.
            if pct == "~" or ranges_overlap(base, base_var, head, head_var):
                continue
            changes.append(
                Change(
                    name=name,
                    base=base,
                    head=head,
                    pct=float(pct.removesuffix("%")),
                    pval=pval,
                )
            )
    if comparisons == 0:
        raise ValueError("No comparable base/head benchmark results were found.")
    return changes


def render_markdown(changes: list[Change], threshold: float) -> str:
    regressions = sorted([c for c in changes if c.pct > 0], key=lambda c: -c.pct)
    improvements = sorted([c for c in changes if c.pct < 0], key=lambda c: c.pct)

    if not regressions and not improvements:
        return "### ✅ No significant performance changes detected\n"

    lines: list[str] = []

    if regressions:
        above = [r for r in regressions if r.pct > threshold]
        if above:
            lines.append(
                f"### ⚠️ {len(regressions)} regression(s) detected (threshold: >{threshold:g}%)\n"
            )
        else:
            lines.append(
                f"### 📊 {len(regressions)} minor regression(s) (all within {threshold:g}% threshold)\n"
            )

        lines.append("| Benchmark | Base | Head | Change | p-value |")
        lines.append("|-----------|------|------|--------|---------|")
        for r in regressions:
            change = f"+{r.pct:.2f}%"
            if r.pct > threshold:
                change = f"**{change}** 🔴"
            lines.append(f"| `{r.name}` | {r.base} | {r.head} | {change} | {r.pval} |")
        lines.append("")

    if improvements:
        lines.append(f"<details>")
        lines.append(f"<summary>✅ {len(improvements)} improvement(s)</summary>\n")
        lines.append("| Benchmark | Base | Head | Change | p-value |")
        lines.append("|-----------|------|------|--------|---------|")
        for imp in improvements:
            lines.append(
                f"| `{imp.name}` | {imp.base} | {imp.head} | {imp.pct:.2f}% | {imp.pval} |"
            )
        lines.append("")
        lines.append("</details>")
        lines.append("")

    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", help="Path to benchstat output file")
    parser.add_argument(
        "--threshold",
        type=float,
        default=5,
        help="Regression percentage threshold to flag as failure (default: 5)",
    )
    args = parser.parse_args()

    try:
        changes = parse_benchstat(args.input)
    except ValueError as err:
        print(f"### ❌ Benchmark comparison unavailable\n\n{err}")
        sys.exit(2)
    summary = render_markdown(changes, args.threshold)
    print(summary)

    # Exit 1 if any regression exceeds the threshold
    regressions_above_threshold = [c for c in changes if c.pct > args.threshold]
    if regressions_above_threshold:
        print(f"\nFailed: {len(regressions_above_threshold)} benchmark(s) regressed by more than {args.threshold:g}%:")
        for c in sorted(regressions_above_threshold, key=lambda c: -c.pct):
            print(f"  {c.name}: {c.base} -> {c.head} (+{c.pct:.2f}%)")
        sys.exit(1)


if __name__ == "__main__":
    main()
