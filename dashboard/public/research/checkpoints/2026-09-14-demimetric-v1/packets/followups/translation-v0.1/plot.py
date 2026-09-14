#!/usr/bin/env python3
"""Render stored exact-check results; Matplotlib is optional, plot values rounded."""
import json
from fractions import Fraction
from pathlib import Path

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt


def main():
    root = Path(__file__).resolve().parent
    data = json.loads((root / "plot-data.json").read_text())
    if data["schema"] != "zerone-translation-plot-data/v1" or not data["plot_values_only"]:
        raise ValueError("Unexpected plot data")
    plt.rcParams.update({"font.size": 10, "svg.hashsalt": "translation-v0.1",
                         "svg.fonttype": "none", "axes.spines.top": False,
                         "axes.spines.right": False})
    fig, axes = plt.subplots(1, 2, figsize=(10, 4.1), sharey=True)
    colors = ["#0072B2", "#D55E00", "#009E73", "#CC79A7"]
    policies = ["raw_origin_damping", "affine_combination_candidate"]
    titles = ["Original: physical answer changes", "Candidate: translated paths coincide"]
    for ax, policy, title in zip(axes, policies, titles):
        series = [s for s in data["series"] if s["policy"] == policy]
        for index, (item, color) in enumerate(zip(series, colors)):
            points = item["physical_pulled_back_positions"]
            shift = str(Fraction(item["shift"]))
            ax.plot([p["n"] for p in points],
                    [float(p["value_plot_only"]) for p in points],
                    color=color, linewidth=1.7, label=f"origin t = {shift}",
                    marker="o" if policy == policies[1] else None,
                    markersize=3, markevery=(10 + index * 6, 28))
        ax.axhline(1, color="#333333", linestyle="--", linewidth=1,
                   label="required physical target = 1")
        ax.set(title=title, xlabel="Iterate index n (128 updates)", xlim=(0, 129),
               ylim=(-2.15, 1.75))
        ax.grid(alpha=0.18)
    axes[0].set_ylabel("Physical position after converting back: yₙ + t")
    handles, labels = axes[0].get_legend_handles_labels()
    fig.legend(handles, labels, loc="lower center", ncol=3, frameon=False,
               bbox_to_anchor=(0.5, -0.005))
    fig.suptitle("Same identity problem, four coordinate origins", fontsize=13)
    fig.subplots_adjust(left=0.09, right=0.98, bottom=0.25, top=0.82, wspace=0.12)
    fig.savefig(root / "translation-check.svg", metadata={"Date": None,
                "Description": "Finite exact-check trajectories rendered with rounded values. "
                               "Analytic limits and scope are explained in NOTE.md."})
    svg = root / "translation-check.svg"
    svg.write_text("\n".join(line.rstrip() for line in svg.read_text().splitlines()) + "\n")
    plt.close(fig)


if __name__ == "__main__":
    main()
