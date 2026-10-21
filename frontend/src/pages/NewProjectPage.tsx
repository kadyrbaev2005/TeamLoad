import { useQueryClient } from "@tanstack/react-query";
import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { ApiError, api, errorMessage } from "../api/client";
import { keys, useCreateProject } from "../api/hooks";
import type { Plan } from "../api/types";
import { Counter } from "../components/Field";
import { Icon } from "../components/Icon";
import { useAiOverlay } from "../components/LoadingOverlay";
import { useToast } from "../components/Toast";
import { Notice, PageHead } from "../components/ui";
import { AppShell } from "../layout/AppShell";
import { designTeamSteps } from "../lib/aiSteps";
import { addDaysISO, todayISO } from "../lib/format";
import { validateNewProject } from "../lib/validate";

export function NewProjectPage() {
  const [f, setF] = useState({ title: "", description: "", deadline: "", size: 5, context: "" });
  const [err, setErr] = useState("");
  const create = useCreateProject();
  const { run, overlay } = useAiOverlay();
  const navigate = useNavigate();
  const toast = useToast();
  const qc = useQueryClient();
  const minDeadline = addDaysISO(todayISO(), 7);
  const set = (k: keyof typeof f) => (v: string | number) => setF((s) => ({ ...s, [k]: v }));

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const problem = validateNewProject(f);
    setErr(problem ?? "");
    if (problem) return;
    let projectId: string;
    try {
      const b = await create.mutateAsync({ title: f.title.trim(), description: f.description.trim(), deadline: f.deadline, team_size: f.size, context: f.context.trim() });
      projectId = b.project.id;
    } catch (e2) {
      const fields = e2 instanceof ApiError ? Object.values(e2.fields) : [];
      setErr(fields[0] ?? errorMessage(e2));
      return;
    }
    try {
      const res = await run("Designing your team", designTeamSteps(f.deadline, f.size), () =>
        api<{ plan: Plan }>("POST", `/projects/${projectId}/plan/generate`, { note: "" }),
      );
      qc.setQueryData(keys.plan(projectId), res.plan);
      void qc.invalidateQueries({ queryKey: keys.project(projectId) });
      navigate(`/p/${projectId}/plan`);
      toast("Role plan ready: review and approve it");
    } catch {
      // The project is saved; the plan page shows the AI error with Retry.
      navigate(`/p/${projectId}/plan`, { state: { aiFailed: true } });
    }
  };

  const seats = Array.from({ length: f.size }, (_, i) => (
    <span key={i} className={`seat${i === 0 ? " you" : ""}`} title={i === 0 ? "You" : "Open seat"} />
  ));

  return (
    <AppShell active="projects">
      <PageHead eyebrow="New project" title="Describe your project" sub="The AI uses this to decide which roles your team needs." />
      <div className="split">
        <form className="panel pad stack" onSubmit={onSubmit} noValidate>
          <div className="field">
            <label htmlFor="nTitle">Project title</label>
            <input className="input" id="nTitle" maxLength={100} value={f.title} onChange={(e) => set("title")(e.target.value)} />
          </div>
          <div className="field">
            <label htmlFor="nDesc">What are you building?</label>
            <textarea
              className="textarea"
              id="nDesc"
              style={{ minHeight: 170 }}
              maxLength={3000}
              value={f.description}
              onChange={(e) => set("description")(e.target.value)}
              aria-describedby="nDescHint nCount"
            />
            <div className="row" style={{ justifyContent: "space-between" }}>
              <span className="hint" id="nDescHint">
                Goal, main features, who uses it, any known technology.
              </span>
              <span id="nCount">
                <Counter value={f.description} max={3000} min={100} suffix=" · min 100" />
              </span>
            </div>
          </div>
          <div className="grid2">
            <div className="field">
              <label htmlFor="nDeadline">Deadline</label>
              <input className="input" id="nDeadline" type="date" min={minDeadline} value={f.deadline} onChange={(e) => set("deadline")(e.target.value)} />
              <span className="hint">At least 7 days from today.</span>
            </div>
            <div className="field">
              <label id="nSizeLabel">Team size</label>
              <div className="row">
                <div className="stepper" role="group" aria-labelledby="nSizeLabel">
                  <button type="button" aria-label="Fewer members" onClick={() => set("size")(Math.max(3, f.size - 1))}>
                    −
                  </button>
                  <output id="nSize" aria-live="polite">
                    {f.size}
                  </output>
                  <button type="button" aria-label="More members" onClick={() => set("size")(Math.min(6, f.size + 1))}>
                    +
                  </button>
                </div>
                <span className="hint">3–6, including you</span>
              </div>
              <div className="seats" aria-hidden="true">
                {seats}
              </div>
            </div>
          </div>
          <div className="field">
            <label htmlFor="nCtx">
              Course or context{" "}
              <span className="muted" style={{ fontWeight: 400 }}>
                (optional)
              </span>
            </label>
            <input className="input" id="nCtx" maxLength={100} value={f.context} onChange={(e) => set("context")(e.target.value)} />
          </div>
          {err && (
            <p className="small" role="alert" style={{ color: "var(--bad)" }}>
              {err}
            </p>
          )}
          <div className="row">
            <button className="btn primary" type="submit" disabled={create.isPending}>
              <Icon name="spark" />
              Generate role plan
            </button>
            <Link className="btn ghost" to="/projects">
              Cancel
            </Link>
          </div>
        </form>
        <aside className="stack">
          <div className="panel pad stack" style={{ gap: 10 }}>
            <h3>What makes a good description</h3>
            <ul className="goodlist">
              <li>What the product does and for whom.</li>
              <li>The 3–5 main features.</li>
              <li>Anything special: a map, payments, AI, hardware, analytics.</li>
              <li>Technology you already chose, if any.</li>
            </ul>
          </div>
          <Notice tone="ai">
            <b>What happens next.</b> The AI proposes {f.size} roles, one per seat. You can edit or regenerate them before inviting your team.
          </Notice>
        </aside>
      </div>
      {overlay}
    </AppShell>
  );
}
