import { useState } from "react";
import { Link, useLocation } from "react-router";
import { errorMessage } from "../api/client";
import { useApprovePlan, useGeneratePlan, usePlan, useSavePlan, useStandardPlan } from "../api/hooks";
import type { Plan, ProjectBundle, Seat } from "../api/types";
import { Icon } from "../components/Icon";
import { useAiOverlay } from "../components/LoadingOverlay";
import { ConfirmModal } from "../components/Modal";
import { useCopy, useToast } from "../components/Toast";
import { Button, Chip, EffortBar, Legend, Notice, PageHead, PageSkeleton, Swatch } from "../components/ui";
import { ProjectPage } from "../layout/ProjectPage";
import { designTeamSteps, redesignSteps } from "../lib/aiSteps";
import { fmtDate } from "../lib/format";
import { MSG } from "../lib/validate";

type Draft = Pick<Seat, "id" | "name" | "summary" | "effort_share">;

function Editable({ as: Tag, value, editing, onChange, label }: {
  as: "h3" | "p";
  value: string;
  editing: boolean;
  onChange: (v: string) => void;
  label: string;
}) {
  if (!editing) return <Tag className="editable">{value}</Tag>;
  return (
    <Tag
      className="editable"
      contentEditable
      suppressContentEditableWarning
      role="textbox"
      aria-label={label}
      tabIndex={0}
      onBlur={(e) => onChange(e.currentTarget.textContent?.trim() ?? "")}
      onKeyDown={(e) => {
        if (e.key === "Enter") {
          e.preventDefault();
          e.currentTarget.blur();
        }
      }}
    >
      {value}
    </Tag>
  );
}

function SeatCard({ s, i, total, draft, onDraft }: { s: Seat; i: number; total: number; draft?: Draft; onDraft: (d: Draft) => void }) {
  const editing = !!draft;
  const name = draft?.name ?? s.name;
  return (
    <article className="panel slot">
      <div className="top">
        <div className="stack" style={{ gap: 6 }}>
          <span className="num">
            SEAT {i + 1} OF {total}
          </span>
          <Editable as="h3" value={name} editing={editing} label={`Role name for seat ${i + 1}`} onChange={(v) => draft && onDraft({ ...draft, name: v })} />
        </div>
        <span className="swatch" style={{ background: `var(${s.color})`, width: 14, height: 14, marginTop: 4 }} />
      </div>
      <Editable
        as="p"
        value={draft?.summary ?? s.summary}
        editing={editing}
        label={`Summary for ${name}`}
        onChange={(v) => draft && onDraft({ ...draft, summary: v })}
      />
      <div className="stack" style={{ gap: 6 }}>
        <span className="eyebrow">Responsibilities</span>
        <ul>
          {s.responsibilities.map((r) => (
            <li key={r}>{r}</li>
          ))}
        </ul>
      </div>
      <div className="stack" style={{ gap: 6 }}>
        <span className="eyebrow">Skills needed</span>
        <div className="skills">
          {s.skills.map((k) => (
            <span className="chip line" key={k}>
              {k}
            </span>
          ))}
        </div>
      </div>
      <div className="share">
        <span className="eyebrow">Effort share</span>
        {draft ? (
          <>
            <input
              className="input"
              type="number"
              min={5}
              max={60}
              step={1}
              value={Number.isNaN(draft.effort_share) ? "" : draft.effort_share}
              onChange={(e) => onDraft({ ...draft, effort_share: Math.round(Number(e.target.value)) })}
              aria-label={`Effort share for ${name}`}
            />
            <span className="small">%</span>
          </>
        ) : (
          <b className="mono">{s.effort_share}%</b>
        )}
      </div>
      <p className="why">
        <b>Why this project needs it.</b> {s.why}
      </p>
    </article>
  );
}

function PlanView({ b, plan }: { b: ProjectBundle; plan: Plan }) {
  const p = b.project;
  const owner = b.is_owner;
  const [drafts, setDrafts] = useState<Draft[] | null>(null);
  const [note, setNote] = useState("");
  const [confirm, setConfirm] = useState<null | { note: string }>(null);
  const [error, setError] = useState("");
  const save = useSavePlan(p.id);
  const approve = useApprovePlan(p.id);
  const generate = useGeneratePlan(p.id);
  const { run, overlay } = useAiOverlay();
  const toast = useToast();
  const copy = useCopy();

  const seats = plan.seats;
  const shares = seats.map((s, i) => drafts?.[i]?.effort_share ?? s.effort_share);
  const sum = shares.reduce((a, v) => a + (Number.isFinite(v) ? v : 0), 0);
  const approved = plan.status === "approved";
  const editing = drafts !== null;

  const doRegenerate = async (n: string) => {
    setConfirm(null);
    setError("");
    try {
      await run("Redesigning roles", redesignSteps(n !== "", seats.length), () => generate.mutateAsync(n));
      setNote("");
      toast("New draft ready. Review it, then approve.");
    } catch (e) {
      setError(errorMessage(e));
    }
  };
  const askRegenerate = (n: string) => (plan.has_matching || plan.has_raci ? setConfirm({ note: n }) : void doRegenerate(n));

  const onSave = () =>
    drafts &&
    save.mutate(drafts, {
      onSuccess: () => {
        setDrafts(null);
        toast("Role plan saved");
      },
      onError: (e) => setError(errorMessage(e)),
    });

  const actions = !owner ? null : editing ? (
    <>
      <Button variant="primary" icon="check" disabled={sum !== 100 || save.isPending} onClick={onSave}>
        Save changes
      </Button>
      <Button onClick={() => setDrafts(null)}>Cancel</Button>
    </>
  ) : (
    <>
      <Button icon="pen" onClick={() => setDrafts(seats.map((s) => ({ id: s.id, name: s.name, summary: s.summary, effort_share: s.effort_share })))}>
        Edit
      </Button>
      <Button icon="refresh" onClick={() => askRegenerate("")}>
        Regenerate
      </Button>
      {!approved && (
        <Button
          variant="primary"
          icon="check"
          disabled={sum !== 100 || approve.isPending}
          onClick={() => approve.mutate(undefined, { onSuccess: () => toast("Plan approved. Matching is now available."), onError: (e) => setError(errorMessage(e)) })}
        >
          Approve plan
        </Button>
      )}
    </>
  );

  const sub =
    plan.source === "standard"
      ? `Built from the standard role catalogue for ${p.team_size} seats. Deadline ${fmtDate(p.deadline)}.`
      : `Designed by AI from your description, deadline (${fmtDate(p.deadline)}) and team size.`;

  return (
    <>
      <PageHead eyebrow="Role plan" title={`${seats.length} roles for ${p.title}`} sub={sub} actions={actions} />
      <div className="stack" style={{ gap: 20 }}>
        {error && <Notice tone="bad">{error}</Notice>}
        <div className="panel pad stack" style={{ gap: 12 }}>
          <div className="row" style={{ justifyContent: "space-between" }}>
            <div className="statusline">
              {approved ? (
                <Chip tone="ok" icon="check">
                  Approved
                </Chip>
              ) : (
                <Chip tone="warn" dot>
                  Draft
                </Chip>
              )}
              {plan.source === "standard" ? (
                <Chip tone="line">Standard plan</Chip>
              ) : (
                <Chip tone="ai" icon="spark">
                  Generated by AI
                </Chip>
              )}
              {owner && <span className="small muted">Everything here can be edited.</span>}
            </div>
            <span className={`small mono${sum !== 100 ? "" : " muted"}`} style={sum !== 100 ? { color: "var(--bad)", fontWeight: 700 } : undefined} aria-live="polite">
              Effort total {sum}%{sum !== 100 ? " · must be 100%" : ""}
            </span>
          </div>
          <EffortBar
            label="Effort share by role"
            parts={seats.map((s, i) => ({ pct: shares[i] ?? 0, color: `var(${s.color})`, title: `${drafts?.[i]?.name ?? s.name} ${shares[i]}%` }))}
          />
          <Legend>
            {seats.map((s, i) => (
              <span key={s.id}>
                <Swatch color={s.color} />
                {drafts?.[i]?.name ?? s.name} <b className="mono">{shares[i]}%</b>
              </span>
            ))}
          </Legend>
        </div>
        <div className="slots">
          {seats.map((s, i) => (
            <SeatCard
              key={s.id}
              s={s}
              i={i}
              total={seats.length}
              draft={drafts?.[i]}
              onDraft={(d) => setDrafts((all) => all && all.map((x, k) => (k === i ? d : x)))}
            />
          ))}
        </div>
        <div className="grid2">
          {owner && (
            <div className="panel pad stack" style={{ gap: 10 }}>
              <h3>Not quite right?</h3>
              <p className="small muted">Tell the AI what to change and it will redesign the roles. Your current plan stays until the new one is ready.</p>
              <textarea
                className="textarea"
                style={{ minHeight: 70 }}
                maxLength={500}
                value={note}
                onChange={(e) => setNote(e.target.value)}
                aria-label="Note for the AI"
                placeholder="For example: we need more focus on design, merge the data role into backend"
              />
              <div>
                <Button icon="refresh" onClick={() => askRegenerate(note.trim())}>
                  Regenerate with this note
                </Button>
              </div>
            </div>
          )}
          <div className="panel pad stack" style={{ gap: 10 }}>
            <h3>Invite your team</h3>
            <p className="small muted">
              {approved ? `Share the code or link. The project is full at ${p.team_size} members.` : "Members can join now. Matching starts after you approve the plan."}
            </p>
            <div className="invite">
              <span className="code">{p.invite_code}</span>
              <Button size="sm" icon="copy" onClick={() => copy(p.invite_code)}>
                Copy code
              </Button>
              <Button size="sm" icon="copy" onClick={() => copy(p.invite_link)}>
                Copy link
              </Button>
            </div>
            <p className="small">
              <b className="mono">
                {b.members.length} / {p.team_size}
              </b>{" "}
              joined · <Link to={`/p/${p.id}/roles`}>Go to team roles</Link>
            </p>
          </div>
        </div>
      </div>
      {confirm && (
        <ConfirmModal
          title="Regenerate the role plan?"
          message="This replaces the roles. Matching and RACI will be cleared."
          confirmLabel="Regenerate"
          danger
          onConfirm={() => void doRegenerate(confirm.note)}
          onCancel={() => setConfirm(null)}
        />
      )}
      {overlay}
    </>
  );
}

/** No seats yet: first generation failed or never ran. */
function NoPlan({ b }: { b: ProjectBundle }) {
  const p = b.project;
  const loc = useLocation();
  const [failed, setFailed] = useState<string>((loc.state as { aiFailed?: boolean } | null)?.aiFailed ? MSG.aiUnavailable : "");
  const generate = useGeneratePlan(p.id);
  const standard = useStandardPlan(p.id);
  const { run, overlay } = useAiOverlay();
  const toast = useToast();

  const retry = async () => {
    setFailed("");
    try {
      await run("Designing your team", designTeamSteps(p.deadline, p.team_size), () => generate.mutateAsync(""));
      toast("Role plan ready: review and approve it");
    } catch (e) {
      setFailed(errorMessage(e));
    }
  };
  const useStandard = () =>
    standard.mutate(undefined, { onSuccess: () => toast("Standard plan ready: edit it, then approve"), onError: (e) => setFailed(errorMessage(e)) });

  return (
    <>
      <PageHead eyebrow="Role plan" title={`Roles for ${p.title}`} sub={`Deadline ${fmtDate(p.deadline)} · ${p.team_size} seats.`} />
      {!b.is_owner ? (
        <div className="panel pad empty">
          <h3>No role plan yet</h3>
          <p className="muted">The project owner has not created the role plan yet. You can fill in your profile meanwhile.</p>
          <Link className="btn" to="/profile">
            Open my profile
          </Link>
        </div>
      ) : (
        <div className="stack">
          {failed && (
            <Notice tone="bad">
              <div className="stack" style={{ gap: 10 }}>
                <span>{failed}</span>
                <div className="row">
                  <Button size="sm" icon="refresh" onClick={() => void retry()}>
                    Retry
                  </Button>
                  <Button size="sm" onClick={useStandard} disabled={standard.isPending}>
                    Use a standard plan
                  </Button>
                </div>
              </div>
            </Notice>
          )}
          {!failed && (
            <div className="panel pad empty">
              <span className="chip ai">
                <Icon name="spark" />
                Role plan
              </span>
              <h2>Your project has no role plan yet.</h2>
              <p className="muted">The AI proposes {p.team_size} roles, one per seat. You can edit or regenerate them before inviting your team.</p>
              <div className="row">
                <Button variant="primary" icon="spark" onClick={() => void retry()}>
                  Generate role plan
                </Button>
                <Button onClick={useStandard} disabled={standard.isPending}>
                  Use a standard plan
                </Button>
              </div>
            </div>
          )}
        </div>
      )}
      {overlay}
    </>
  );
}

function PlanScreen({ b }: { b: ProjectBundle }) {
  const plan = usePlan(b.project.id);
  if (plan.error) return <Notice tone="bad">{errorMessage(plan.error)}</Notice>;
  if (!plan.data) return <PageSkeleton blocks={3} />;
  if (plan.data.seats.length === 0) return <NoPlan b={b} />;
  return <PlanView b={b} plan={plan.data} />;
}

export function PlanPage() {
  return <ProjectPage active="plan">{(b) => <PlanScreen b={b} />}</ProjectPage>;
}
