"use client";

import type { WorkflowStage } from "@/components/dashboard/types";

type WorkflowRailProps = {
  stages: WorkflowStage[];
  onSelectStage: (stageValue: string) => void;
};

const chapterNumerals = ["I", "II", "III", "IV", "V", "VI", "VII", "VIII"];

export function WorkflowRail({ stages, onSelectStage }: WorkflowRailProps) {
  if (stages.length === 0) return null;

  return (
    <nav className="workflow-rail" aria-label="Memoir 工作流">
      {stages.map((stage, index) => (
        <button
          type="button"
          key={stage.value}
          className="workflow-step workflow-step-button"
          data-state={stage.state}
          onClick={() => onSelectStage(stage.value)}
        >
          <span className="workflow-step-number">
            {chapterNumerals[index] ?? index + 1}
          </span>
          <span className="workflow-step-copy">
            <strong>{stage.label}</strong>
            <span>{stage.description}</span>
          </span>
        </button>
      ))}
    </nav>
  );
}
